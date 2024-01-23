package notifications

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	logger "gitlab.cee.redhat.com/service/ocm-common/pkg/ocmlogger"
)

type NotificationSender interface {
	Send(ctx context.Context, payload *NotificationPayload) error
}

type Client struct {
	Config     *ClientConfiguration
	HTTPClient *http.Client

	router *mux.Router
}

type ServiceClient interface {
	// GetServiceName returns the name of the service
	GetServiceName() string
	// GetRouter returns the router with the routes
	GetRouter() *mux.Router
}

func NewClient(ctx context.Context, config *ClientConfiguration, metricsMiddleware *func(client ServiceClient, httpClient *http.Client) *http.Client) (*Client, error) {
	var (
		proxyURL    *url.URL
		certificate tls.Certificate
		err         error
	)

	ulog := logger.NewOCMLogger(ctx)

	if _, err = url.Parse(config.BaseURL); err != nil {
		ulog.Err(err).Error("Notifications base url.Parse")
		return nil, err
	}

	if len(config.ProxyURL) != 0 {
		if proxyURL, err = url.Parse(config.ProxyURL); err != nil {
			ulog.Err(err).Error("Notifications proxy url.Parse")
			return nil, err
		}
	}

	if certificate, err = tls.X509KeyPair([]byte(config.Cert), []byte(config.Key)); err != nil {
		ulog.Err(err).Error("Notifications tls.X509KeyPair")
		return nil, err
	}

	client := &Client{
		Config: config,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{certificate},
				},
			},
		},
	}
	if metricsMiddleware != nil {
		client.HTTPClient = (*metricsMiddleware)(&Client{}, client.HTTPClient)
	}
	return client, nil
}

func (n *Client) Send(ctx context.Context, payload *NotificationPayload) error {
	ulog := logger.NewOCMLogger(ctx)
	body, err := json.Marshal(*payload)
	if err != nil {
		ulog.Err(err).Extra("payload", *payload).Error("Notifications.send: read body")
		return err
	}
	req, err := http.NewRequest(http.MethodPost, n.Config.BaseURL, bytes.NewReader(body))
	if err != nil {
		ulog.Err(err).Extra("payload", *payload).Error("Notifications.send: http.NewRequest")
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := n.HTTPClient.Do(req)
	if err != nil {
		ulog.Err(err).Extra("payload", *payload).Error("Notifications.send: .Do(req)")
		return err
	}
	if res.StatusCode != http.StatusOK {
		ulog.Extra("payload", *payload).Error("Notifications.status: .Do(req): %d", res.StatusCode)
		return errors.New(http.StatusText(res.StatusCode))
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		ulog.Err(err).Extra("payload", *payload).Error("Notifications.send: read body")
		return err
	}
	type responseType struct {
		Result  string `json:"result"`
		Details string `json:"details,omitempty"`
	}
	var response responseType
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		ulog.Err(err).Extra("payload", *payload).Extra("response-body", resBody).Error("Notifications.send: response body is not JSON")
		return nil
	}
	if response.Result == "error" {
		ulog.Extra("payload", *payload).Extra("response", response).Error("Notifications Email received error")
		return errors.New(response.Details)
	}

	ulog.Extra("payload", *payload).Extra("response-body", resBody).Extra("response status", res.StatusCode).Extra("response", response).Info("Notifications Email received success")
	return nil
}

func (n *Client) GetServiceName() string {
	return "notifications"
}

func (n *Client) GetRouter() *mux.Router {
	if n.router == nil {
		n.router = mux.NewRouter()
		mainRoute := n.router.PathPrefix("/api/notifications-gw/notifications").Subrouter()
		mainRoute.Path("/").Methods(http.MethodPost)
	}
	return n.router
}

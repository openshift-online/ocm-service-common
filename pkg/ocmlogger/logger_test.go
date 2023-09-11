package ocmlogger

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("logger.Extra", Label("logger"), func() {
	var (
		ulog   OCMLogger
		output bytes.Buffer
	)

	BeforeEach(func() {
		ulog = NewOCMLogger(context.Background())
		SetOutput(&output)
		DeferCleanup(func() {
			SetOutput(os.Stderr)
		})
	})

	Context("basic; no Extra", func() {
		It("ignores Info level (default min level is Warning)", func() {
			ulog.Info("message")

			result := output.String()
			Expect(result).To(Equal(""))
		})

		It("includes just message", func() {
			ulog.Warning("message")

			result := output.String()
			Expect(result).NotTo(ContainSubstring("\"Extra\":{"))
			Expect(result).To(ContainSubstring("\"message\":\"message\""))
		})
	})

	Context("awareness of simple types", func() {
		It("all simple types are in Extra dictionary", func() {
			ulog.Extra("true", true)
			ulog.Extra("false", false)
			ulog.Extra("int", 1)
			ulog.Extra("int8", int8(8))
			ulog.Extra("int16", int16(16))
			ulog.Extra("int32", int32(32))
			ulog.Extra("float32", float32(32.01))
			ulog.Extra("float64", 64.01)
			ulog.Warning("message")

			result := output.String()
			Expect(result).To(ContainSubstring("\"Extra\":{"))
			Expect(result).To(ContainSubstring("\"true\":true"))
			Expect(result).To(ContainSubstring("\"false\":false"))
			Expect(result).To(ContainSubstring("\"int\":1"))
			Expect(result).To(ContainSubstring("\"int8\":8"))
			Expect(result).To(ContainSubstring("\"int16\":16"))
			Expect(result).To(ContainSubstring("\"int32\":32"))
			Expect(result).To(ContainSubstring("\"float32\":32.01"))
			Expect(result).To(ContainSubstring("\"float64\":64.01"))
		})
	})

	Context("setting same key", func() {
		It("overrides value", func() {
			ulog.Extra("key1", 1)
			ulog.Extra("key1", 2)
			ulog.Warning("warning")

			result := output.String()
			Expect(result).To(ContainSubstring("\"key1\":2"))
		})
	})

	Context("complex/nested types", func() {
		It("each will present in output", func() {
			headers1 := http.Header{}
			headers1["Content-Type"] = []string{"application/json"}
			headers1["Content-Length"] = []string{"0"}

			resp1 := http.Response{
				StatusCode: 200,
				Header:     headers1,
			}
			ulog.Extra("resp1", resp1)

			headers2 := http.Header{}
			headers2["Content-Type"] = []string{"application/xml"}
			headers2["Content-Length"] = []string{"100"}
			resp2 := http.Response{
				StatusCode: 404,
				Header:     headers2,
			}
			ulog.Extra("resp2", resp2)

			ulog.Warning("warning")
			result := output.String()
			Expect(result).To(ContainSubstring("\"resp1\":{"))
			Expect(result).To(ContainSubstring("\"resp2\":{"))
			Expect(result).To(ContainSubstring("\"StatusCode\":200"))
			Expect(result).To(ContainSubstring("\"StatusCode\":404"))
			Expect(result).To(ContainSubstring("\"Header\":{\"Content-Length\":[\"0\"],\"Content-Type\":[\"application/json\"]}"))
			Expect(result).To(ContainSubstring("\"Header\":{\"Content-Length\":[\"100\"],\"Content-Type\":[\"application/xml\"]}"))
			Expect(result).To(ContainSubstring("\"StatusCode\":404"))
		})
	})

	Context("Error", func() {
		It("adds error message, sets level to error", func() {
			ulog.Err(fmt.Errorf("error-message"))
			ulog.Error("ERROR")

			result := output.String()
			Expect(result).To(ContainSubstring("\"level\":\"error\","))
			Expect(result).To(ContainSubstring("\"error\":\"error-message\","))
			Expect(result).To(ContainSubstring("\"message\":\"ERROR\""))
		})
	})

	Context("supported context keys are added to output", func() {
		BeforeEach(func() {
			ctx := context.Background()
			getOpId := func(ctx context.Context) string {
				return ctx.Value("opID").(string)
			}
			SetOpIDCallback(getOpId)
			getAccountId := func(ctx context.Context) string {
				return ctx.Value("accountID").(string)
			}
			SetAccountIDCallback(getAccountId)
			getTxId := func(ctx context.Context) int64 {
				return ctx.Value("tx_id").(int64)
			}
			SetTxIDCallback(getTxId)
			ctx = context.WithValue(ctx, "opID", "OpId1")
			ctx = context.WithValue(ctx, "accountID", "AccountID")
			ctx = context.WithValue(ctx, "tx_id", int64(123))
			ulog = NewOCMLogger(ctx)

			DeferCleanup(func() {
				SetOpIDCallback(nil)
				SetAccountIDCallback(nil)
				SetTxIDCallback(nil)
			})
		})

		It("each one is added to output", func() {
			ulog.Warning("warning")

			result := output.String()
			Expect(result).To(ContainSubstring("\"level\":\"warn\""))
			Expect(result).To(ContainSubstring("\"opid\":\"OpId1\""))
			Expect(result).To(ContainSubstring("\"accountID\":\"AccountID\""))
			Expect(result).To(ContainSubstring("\"tx_id\":123"))
		})
	})
})

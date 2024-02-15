package ocmlogger

import (
	"bytes"
	"context"
	"fmt"
	"math"
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
		output = bytes.Buffer{}
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
			ulog.CaptureSentryEvent(false).Error("ERROR")

			result := output.String()
			Expect(result).To(ContainSubstring("\"level\":\"error\","))
			Expect(result).To(ContainSubstring("\"error\":\"error-message\","))
			Expect(result).To(ContainSubstring("\"message\":\"ERROR\""))
		})
	})

	Context("registered context keys are added to output", func() {
		BeforeEach(func() {
			getOpIdFromContext := func(ctx context.Context) any {
				return ctx.Value("opID")
			}
			getTxIdFromContext := func(ctx context.Context) any {
				return ctx.Value("tx_id")
			}

			RegisterExtraDataCallback("opID", getOpIdFromContext)
			RegisterExtraDataCallback("tx_id", getTxIdFromContext)

			ctx := context.Background()

			//lint:ignore SA1029 doesnt matter for a test
			ctx = context.WithValue(ctx, "opID", "OpId1")

			//lint:ignore SA1029 doesnt matter for a test
			ctx = context.WithValue(ctx, "tx_id", int64(123))
			ulog = NewOCMLogger(ctx)

			DeferCleanup(ClearExtraDataCallbacks)
		})

		It("each one is added to output", func() {
			ulog.Warning("warning")

			result := output.String()
			Expect(result).To(ContainSubstring("\"opID\":\"OpId1\""))
			Expect(result).To(ContainSubstring("\"tx_id\":123"))
		})

		It("nil function safe", func() {
			ulog.Warning("warning")
			RegisterExtraDataCallback("nilCallbackFunction", nil)

			result := output.String()
			Expect(result).To(ContainSubstring("\"opID\":\"OpId1\""))
			Expect(result).To(ContainSubstring("\"tx_id\":123"))
			Expect(result).NotTo(ContainSubstring("nilCallbackFunction"))
		})

		It("empty callback map safe", func() {
			ClearExtraDataCallbacks()
			ulog.Warning("warning")

			result := output.String()
			Expect(result).NotTo(ContainSubstring("\"Extra\""))
		})
	})

	Context("Chaos", func() {
		// Notes:
		//	* without locks in place I can reliably produce concurrency issues with as few as 50 iterations
		// 	* 10000 iterations takes about 0.1 seconds on my laptop
		// 	* not advised to crank this too high, above 1000000 on my laptop used >100% cpu and 40 gigs of ram
		//    and weird stuff started to happen before go shot itself to save the system
		maxChaos := 10000
		It("Extra() is thread safe", func() {
			parallelLog := NewOCMLogger(context.Background())
			for i := 0; i < maxChaos; i++ {
				go func(i int) {
					parallelLog.Extra("i", i).Info("Extra() %d", i)
				}(i)
			}
		})
		It("AdditionalCallLevelSkips() is thread safe", func() {
			parallelLog := NewOCMLogger(context.Background())
			for i := 0; i < maxChaos; i++ {
				go func(i int) {
					parallelLog.AdditionalCallLevelSkips(0).Info("AdditionalCallLevelSkips() %d", i)
				}(i)
			}
		})
		It("CaptureSentryEvent() is thread safe", func() {
			parallelLog := NewOCMLogger(context.Background())
			for i := 0; i < maxChaos; i++ {
				go func(i int) {
					parallelLog.CaptureSentryEvent(false).Info("CaptureSentryEvent() %d", i)
				}(i)
			}
		})
		It("Err() is thread safe", func() {
			parallelLog := NewOCMLogger(context.Background())
			for i := 0; i < maxChaos; i++ {
				go func(i int) {
					parallelLog.Err(fmt.Errorf("err %d", i)).Error("Err() %d", i)
				}(i)
			}
		})
		It("Lots of extras and an error for fun", func() {
			parallelLog := NewOCMLogger(context.Background())
			maxExtras := int(math.Sqrt(math.Max(float64(maxChaos*maxChaos), 100000)))
			for i := 0; i < maxChaos; i++ {
				go func(i int) {
					for j := 0; j < maxExtras; j++ {
						parallelLog.Extra(fmt.Sprintf("%d-%d", i, j), i+j)
					}
					parallelLog.Err(fmt.Errorf("err %d", i)).Error("Lots of extras %d", i)
				}(i)
			}
		})
	})
})

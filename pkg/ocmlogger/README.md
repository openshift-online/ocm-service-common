# OCM LOGER

`glog` package is no longer maintained. `Go 1.21` will include a structured logger based on https://github.com/rs/zerolog. This is a generic implementation of logger based on `zerolog` instead of `glog` in preparation to `go 1.21`.

## Usage

`ocmlogger` may be used everywhere starting from the first line of `main` function.

Default loging level is `Warning` and it can be changed using `-u` flag.

Error and Fatal levels may also be reported to Sentry. To use it one have to ensure `sentry.GetHubFromContext(context)` or `sentry.CurrentHub()` to return a sentry connection.

To cut a long filename of a caller you may use `SetTrimList` to initialize it appropriately for you project (see examples).

#### Examples:

```
import "gitlab.cee.redhat.com/service/ocm-common/ocmlogger"

ocmlog := logger.NewOCMLogger(context.Background())

// Set err and report "error running command" on a FATAL level
var err error
... 
ocmlog.Err(err).Fatal("error running command")

ocmlog.Extra("Cluster transfer id", id).Extra("ClusterUUID", clusterUUID).Extra("Recipient", recipient).Extra("Owner", owner).Error("Cluster transfer debug - Detected Expired Cluster Transfer.")

ocmlog.Extra("response", response).Error("error calling service log")

// To capture output in tests
var output bytes.Buffer
ocmlogger.SetOutput(&output)
defer func() {
    ocmlogger.SetOutput(os.Stderr)
}()
Expect(output.String()).To(ContainSubstring("%s", osl.USER_BANNED_LOG_SUMMARY))
Expect(output.String()).To(ContainSubstring("%s", osl.USER_BANNED_LOG_DESCRIPTION))

ocmlog.SetTrimList([]string{"uhc-account-manager", "pkg"})
```

#### Notes:

1. Try to keep messages constant, use `Extra` to add extra data and `Err` to add error code. This way messages will be grouped in Sentry. 

## Third Party Library Logging - ocm-sdk, REST, etc

TODO: this section will be expanded in the future. If you want raw examples of how AMS bridges different library loggers to UHCLogger see
* [RequestLoggingMiddleware](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/98c1d5d841b06e3b0d5d7bc2d803dad7c0d600b6/pkg/server/logging/request_logging_middleware.go)
* [OcmSdkLogWrapper](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/98c1d5d841b06e3b0d5d7bc2d803dad7c0d600b6/pkg/logger/ocm_sdk_log_wrapper.go)

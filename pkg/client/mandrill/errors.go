package mandrill

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/golang/glog"
)

func checkResponse(c *Client, req *http.Request, resp *http.Response) error {
	if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
		return nil
	}
	buf := new(bytes.Buffer)
	_, buffErr := buf.ReadFrom(resp.Body)
	if buffErr != nil {
		glog.Error("Unable to read response body for following error")
	} else {
		return fmt.Errorf("Mandrill error: %v %v", resp.StatusCode, buf.String())
	}
	return nil
}

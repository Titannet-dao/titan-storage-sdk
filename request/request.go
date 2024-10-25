package request

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/ipfs/boxo/files"
)

type request struct {
	Ctx       context.Context
	ApiBase   string
	Namespace string
	Args      []string
	Opts      map[string]string
	Body      io.Reader
	Headers   http.Header
}

func PostJsonRPC(client *http.Client, url string, in Request, requestHeader http.Header) ([]byte, error) {
	b, err := json.Marshal(&in)
	if err != nil {
		return nil, errors.Errorf("marshalling request: %v", err)
	}

	var out Response
	err = NewBuilder(client, url, "rpc", requestHeader).BodyBytes(b).Exec(context.Background(), &out)
	if err != nil {
		return nil, errors.Errorf("send request: %v", err)
	}

	if out.Error != nil {
		return nil, out.Error
	}

	return json.Marshal(out.Result)
}

func NewRequest(ctx context.Context, url, namespace string, header http.Header) *request {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	return &request{
		Ctx:       ctx,
		ApiBase:   url,
		Namespace: namespace,
		Headers:   header,
	}
}

type trailerReader struct {
	resp *http.Response
}

func (r *trailerReader) Read(b []byte) (int, error) {
	n, err := r.resp.Body.Read(b)
	if err != nil {
		if e := r.resp.Trailer.Get("X-Stream-Error"); e != "" {
			err = errors.New(e)
		}
	}
	return n, err
}

func (r *trailerReader) Close() error {
	return r.resp.Body.Close()
}

type response struct {
	Output io.ReadCloser
	Error  *Error
	Header http.Header
}

func (r *response) Close() error {
	if r.Output != nil {
		// always drain output (response body)
		_, err1 := io.Copy(io.Discard, r.Output)
		err2 := r.Output.Close()
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func (r *response) Decode(dec interface{}) error {
	defer r.Close()
	if r.Error != nil {
		return r.Error
	}

	return json.NewDecoder(r.Output).Decode(dec)
}

type Error struct {
	Namespace string
	Message   string
	Code      int
}

func (e *Error) Error() string {
	var out string
	if e.Namespace != "" {
		out = e.Namespace + ": "
	}
	if e.Code != 0 {
		out = fmt.Sprintf("%s%d: ", out, e.Code)
	}
	return out + e.Message
}

func (r *request) Send(c *http.Client, method string) (*response, error) {
	url := r.getURL()
	req, err := http.NewRequest(method, url, r.Body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(r.Ctx)

	// Add any headers that were supplied via the Builder.
	req.Header = r.Headers.Clone()

	if fr, ok := r.Body.(*files.MultiFileReader); ok {
		req.Header.Set("Content-Type", "multipart/form-data; boundary="+fr.Boundary())
		req.Header.Set("Content-Disposition", "form-data; name=\"files\"")
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	nresp := new(response)
	nresp.Header = resp.Header.Clone()

	contentType := resp.Header.Get("Content-Type")
	parts := strings.Split(contentType, ";")
	contentType = parts[0]

	nresp.Output = &trailerReader{resp}
	if resp.StatusCode >= http.StatusBadRequest {
		e := &Error{
			Namespace: r.Namespace,
		}

		switch {
		case resp.StatusCode == http.StatusNotFound:
			e.Message = "endpoint not found"
		case contentType == "text/plain":
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning! response (%d) read error: %s\n", resp.StatusCode, err)
			}
			e.Message = string(out)
		case contentType == "application/json":
			if err = json.NewDecoder(resp.Body).Decode(e); err != nil {
				fmt.Fprintf(os.Stderr, "warning! response (%d) unmarshall error: %s\n", resp.StatusCode, err)
			}
		default:
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "response (%d) read error: %s\n", resp.StatusCode, err)
			}
			e.Message = string(out)
		}
		nresp.Error = e
		nresp.Output = nil

		// drain body and close
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	return nresp, nil
}

func (r *request) getURL() string {
	values := make(url.Values)
	for _, arg := range r.Args {
		values.Add("arg", arg)
	}
	for k, v := range r.Opts {
		values.Add(k, v)
	}

	if r.Namespace == "rpc" {
		return r.ApiBase
	}

	return fmt.Sprintf("%s/%s?%s", r.ApiBase, r.Namespace, values.Encode())
}

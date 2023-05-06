package request

import (
	"io/ioutil"
	"net/http"
	"time"
)

type DefaultHttpRequest struct {
	Endpoint string
	Path     string
	Method   string
	Token    string
}

func (req *DefaultHttpRequest) DoGetRequest() ([]byte, error) {
	url := "https://" + req.Endpoint + req.Path
	request, err := http.NewRequest(req.Method, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+req.Token)

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	return body, err
}

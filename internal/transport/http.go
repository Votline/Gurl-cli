package transport

import (
	"go.uber.org/zap"

	"Gurl-cli/internal/config"
)

func (c *HTTPClient) Get(cfg *config.HTTPConfig) (Result, error) {
	req, err := c.prepareRequest(cfg)
	if err != nil {
		c.log.Error("Prepare request error", zap.Error(err))
		return Result{}, err
	}

	res, err := c.clientDo(req)
	if err != nil {
		c.log.Error("Couldn't get response", zap.Error(err))
		return Result{}, err
	}
	defer res.Body.Close()

	body := c.extBody(res.Body)
	data := c.convData(body, res)

	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func (c *HTTPClient) Post(cfg *config.HTTPConfig) (Result, error) {
	req, err := c.prepareRequest(cfg)
	if err != nil {
		c.log.Error("Prepare request error", zap.Error(err))
		return Result{}, err
	}

	res, err := c.clientDo(req)
	if err != nil {
		c.log.Error("Couldn't get response", zap.Error(err))
		return Result{}, err
	}
	defer res.Body.Close()

	body := c.extBody(res.Body)
	data := c.convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func (c *HTTPClient) Put(cfg *config.HTTPConfig) (Result, error) {
	req, err := c.prepareRequest(cfg)
	if err != nil {
		c.log.Error("Prepare request error", zap.Error(err))
		return Result{}, err
	}
	
	res, err := c.clientDo(req)
	if err != nil {
		c.log.Error("Couldn't get response", zap.Error(err))
		return Result{}, err
	}
	defer res.Body.Close()

	body := c.extBody(res.Body)
	data := c.convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

func (c *HTTPClient) Del(cfg *config.HTTPConfig) (Result, error) {
	req, err := c.prepareRequest(cfg)
	if err != nil {
		c.log.Error("Prepare request error", zap.Error(err))
		return Result{}, err
	}

	res, err := c.clientDo(req)
	if err != nil {
		c.log.Error("Couldn't get response", zap.Error(err))
		return Result{}, err
	}
	defer res.Body.Close()

	body := c.extBody(res.Body)
	data := c.convData(body, res)
	return Result{Raw: res, RawBody: body, JSON: data}, nil
}

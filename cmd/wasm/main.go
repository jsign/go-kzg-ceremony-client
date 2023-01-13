package main

import (
	"fmt"
	"syscall/js"

	"github.com/savid/go-kzg-ceremony-client/contribution"
)

type fn func(this js.Value, args []js.Value) (any, error)

var (
	jsErr     js.Value = js.Global().Get("Error")
	jsPromise js.Value = js.Global().Get("Promise")
)

func asyncFunc(innerFunc fn) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		handler := js.FuncOf(func(_ js.Value, promFn []js.Value) any {
			resolve, reject := promFn[0], promFn[1]

			go func() {
				defer func() {
					if r := recover(); r != nil {
						reject.Invoke(jsErr.New(fmt.Sprint("panic:", r)))
					}
				}()

				res, err := innerFunc(this, args)
				if err != nil {
					reject.Invoke(jsErr.New(err.Error()))
				} else {
					resolve.Invoke(res)
				}
			}()

			return nil
		})

		return jsPromise.New(handler)
	})
}

func contribute(this js.Value, args []js.Value) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("invalid number of arguments passed: %d", len(args))
	}

	c, err := contribution.DecodeContribution([]byte(args[0].String()))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize contribution: %w", err)
	}

	secret := []byte(args[1].String())
	if len(secret) < 1 {
		return nil, fmt.Errorf("invalid secret")
	}

	if err := c.Contribute(secret); err != nil {
		return nil, fmt.Errorf("failed to contribute: %w", err)
	}

	prevC, err := contribution.DecodeContribution([]byte(args[0].String()))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize contribution: %w", err)
	}

	ok, err := c.Verify(prevC)
	if err != nil {
		return nil, fmt.Errorf("failed to verify contribution: %w", err)
	}

	if !ok {
		return nil, fmt.Errorf("invalid contribution")
	}

	jsonString, err := contribution.Encode(c)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize contribution: %w", err)
	}

	return string(jsonString), nil
}

func verify(this js.Value, args []js.Value) (any, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("invalid number of arguments passed: %d", len(args))
	}

	prevC, err := contribution.DecodeContribution([]byte(args[0].String()))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize previous contribution: %w", err)
	}

	updatedC, err := contribution.DecodeContribution([]byte(args[1].String()))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize updated contribution: %w", err)
	}

	ok, err := updatedC.Verify(prevC)
	if err != nil {
		return false, fmt.Errorf("failed to verify contribution: %w", err)
	}

	if !ok {
		return false, fmt.Errorf("invalid contribution")
	}

	return true, nil
}

func main() {
	js.Global().Set("contribute", asyncFunc(contribute))
	js.Global().Set("verify", asyncFunc(verify))
	select {}
}

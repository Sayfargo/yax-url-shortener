package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	core_app "github.com/Sayfargo/yax-url-shortener/internal/core/app"
)

func main() {

	sigCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app := core_app.New()

	if err := app.Run(sigCtx); err != nil {
		fmt.Println(err)
	}
}

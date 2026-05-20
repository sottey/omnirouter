//go:build !darwin

package main

import "context"

var activeApp *App

func setupMenuBar(ctx context.Context) {
}

func setDockVisible(visible bool) {
}

func teardownMenuBar() {
}

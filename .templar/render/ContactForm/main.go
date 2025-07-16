package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	component := ContactForm("Sample Title")
	
	err := component.Render(ctx, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering component: %v\n", err)
		os.Exit(1)
	}
}

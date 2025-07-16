package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	component := Alert("This is sample content for the component preview. Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Sample AlertType")
	
	err := component.Render(ctx, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering component: %v\n", err)
		os.Exit(1)
	}
}

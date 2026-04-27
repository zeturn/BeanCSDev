package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirm(expected string) error {
	fmt.Printf("Type %q to confirm: ", expected)
	reader := bufio.NewReader(os.Stdin)
	got, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(got) != expected {
		return fmt.Errorf("confirmation did not match")
	}
	return nil
}

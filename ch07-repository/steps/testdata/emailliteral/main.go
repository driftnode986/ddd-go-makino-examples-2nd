// わざとコンパイルできないコードである。
package main

import "example.com/trainingbooking/internal/identity/domain"

func main() {
	_ = domain.Email("not-an-email")
	_ = domain.Email{value: "not-an-email"}
	var _ domain.Email = "not-an-email"
}

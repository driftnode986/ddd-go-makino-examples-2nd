// わざとコンパイルできないコードである。
package main

import "example.com/trainingbooking/internal/identity/domain"

// #region call

func registerMember(name string, email domain.Email) {}

func main() {
	name := "山田太郎"
	email, _ := domain.NewEmail("taro@example.com")
	registerMember(email, name) // 引数が逆
}

// #endregion

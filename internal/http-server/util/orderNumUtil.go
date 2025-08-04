package util

import "github.com/ShiraazMoollatjie/goluhn"

// todo -подумать над структурой ответа функции
func ValidateOrderNum(orderNumber string) []string {
	var errs []string
	if orderNumber == "" {
		// todo -возможно нужно выпилить эту проверку
		errs = append(errs, "empty order number")
	}

	err := goluhn.Validate(orderNumber)
	if err != nil {
		errs = append(errs, err.Error())
	}

	return errs
}

package luhn

import (
	"strconv"
	"strings"
)

// Validate проверяет номер по алгоритму Луна
func Validate(number string) bool {
	// Удаляем пробелы
	number = strings.ReplaceAll(number, " ", "")

	// Проверяем, что строка содержит только цифры
	if len(number) == 0 {
		return false
	}

	for _, ch := range number {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Применяем алгоритм Луна
	sum := 0
	isSecond := false

	// Проходим с конца строки
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}

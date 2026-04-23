package util

import "strings"

func NormalizePhone(phone string) string {
	if phone == "" {
		return ""
	}

	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	phone = strings.TrimPrefix(phone, "+")

	if strings.HasPrefix(phone, "880") {
		phone = phone[2:]
	} else if strings.HasPrefix(phone, "88") && len(phone) > 10 {
		phone = phone[2:]
	}

	if len(phone) == 10 && !strings.HasPrefix(phone, "0") {
		phone = "0" + phone
	}

	return phone
}

func IsValidBDPhone(phone string) bool {
	if len(phone) != 11 || !strings.HasPrefix(phone, "01") {
		return false
	}

	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

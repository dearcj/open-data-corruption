package miner

func getDigitsShortener() map[string]int {
	var digits = make(map[string]int)
	digits["п'ятдесяти"] = 50
	digits["сорока"] = 40
	digits["тридцяти"] = 30
	digits["десяти"] = 10
	digits["п'яти"] = 5
	return digits
}

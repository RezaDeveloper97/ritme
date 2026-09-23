package enums

// Hand-ported behaviour of the daily-log / profile enums.

// Score is the normalised 0-100 energy score used for weekly aggregation and charts.
// PHP: backend/app/Enums/EnergyLevel.php:41.
func (e EnergyLevel) Score() int {
	switch e {
	case EnergyLevelVeryLow:
		return 15
	case EnergyLevelLow:
		return 35
	case EnergyLevelMedium:
		return 60
	case EnergyLevelHigh:
		return 80
	case EnergyLevelVeryHigh:
		return 100
	}
	return 0
}

// BmiCategoryFromBmi classifies a BMI into its WHO band; a boundary value lands in the higher band.
// PHP: backend/app/Enums/BmiCategory.php:23.
func BmiCategoryFromBmi(bmi float64) BmiCategory {
	switch {
	case bmi < 18.5:
		return BmiCategoryUnderweight
	case bmi < 25.0:
		return BmiCategoryNormal
	case bmi < 30.0:
		return BmiCategoryOverweight
	}
	return BmiCategoryObese
}

// IsPresent reports clotting actually recorded (none is not). PHP: backend/app/Enums/ClotsAmount.php:37.
func (e ClotsAmount) IsPresent() bool {
	return e != ClotsAmountNone
}

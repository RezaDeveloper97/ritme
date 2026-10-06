package analysis

// IsMoodKey reports whether a symptom key is a mood (mood.moods.*): the health record's «علائم پرتکرار» lists
// physical symptoms only, like the symptom patterns (bloom B-N6-03).
func IsMoodKey(key string) bool { return isMoodKey(key) }

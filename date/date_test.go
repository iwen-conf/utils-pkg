package date

import (
	"testing"
	"time"
)

func TestWeekdayName(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected int
	}{
		{
			name:     "Monday",
			date:     time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC),
			expected: 2,
		},
		{
			name:     "Sunday",
			date:     time.Date(2024, 3, 17, 0, 0, 0, 0, time.UTC),
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WeekdayName(tt.date); got != tt.expected {
				t.Errorf("WeekdayName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetWorkdayCount(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		expected  int
	}{
		{
			name:      "One week",
			startDate: time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC), // Monday
			endDate:   time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC), // Friday
			expected:  5,
		},
		{
			name:      "Including weekend",
			startDate: time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC), // Monday
			endDate:   time.Date(2024, 3, 24, 0, 0, 0, 0, time.UTC), // Sunday
			expected:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetWorkdayCount(tt.startDate, tt.endDate); got != tt.expected {
				t.Errorf("GetWorkdayCount() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsWeekend(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "Saturday",
			date:     time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Sunday",
			date:     time.Date(2024, 3, 17, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Monday",
			date:     time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWeekend(tt.date); got != tt.expected {
				t.Errorf("IsWeekend() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	date := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "YYYY-MM-DD",
			format:   "YYYY-MM-DD",
			expected: "2024-03-18",
		},
		{
			name:     "YYYY/MM/DD",
			format:   "YYYY/MM/DD",
			expected: "2024/03/18",
		},
		{
			name:     "DD/MM/YYYY",
			format:   "DD/MM/YYYY",
			expected: "18/03/2024",
		},
		{
			name:     "YYYY年MM月DD日",
			format:   "YYYY年MM月DD日",
			expected: "2024年03月18日",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDate(date, tt.format); got != tt.expected {
				t.Errorf("FormatDate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		birthDate time.Time
		expected  int
	}{
		{
			name:      "20 years ago",
			birthDate: now.AddDate(-20, 0, 0),
			expected:  20,
		},
		{
			name:      "Not birthday yet this year",
			birthDate: time.Date(now.Year()-20, 12, 31, 0, 0, 0, 0, time.UTC),
			expected:  19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAge(tt.birthDate); got != tt.expected {
				t.Errorf("GetAge() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsLeapYear(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		expected bool
	}{
		{
			name:     "Leap year divisible by 4",
			year:     2024,
			expected: true,
		},
		{
			name:     "Non-leap year",
			year:     2023,
			expected: false,
		},
		{
			name:     "Century year not divisible by 400",
			year:     1900,
			expected: false,
		},
		{
			name:     "Century year divisible by 400",
			year:     2000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLeapYear(tt.year); got != tt.expected {
				t.Errorf("IsLeapYear() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetWeekdayInRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		weekday   int
		expected  int
	}{
		{
			name:      "One month Mondays",
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			weekday:   1, // Monday
			expected:  4, // 4 Mondays in March 2024
		},
		{
			name:      "Invalid weekday",
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			weekday:   7, // Invalid
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWeekdayInRange(tt.startDate, tt.endDate, tt.weekday)
			if len(got) != tt.expected {
				t.Errorf("GetWeekdayInRange() got %v days, want %v days", len(got), tt.expected)
			}
		})
	}
}

func TestGetMonthFirstAndLastDay(t *testing.T) {
	tests := []struct {
		name          string
		date          time.Time
		expectedFirst time.Time
		expectedLast  time.Time
	}{
		{
			name:          "March 2024",
			date:          time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			expectedFirst: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedLast:  time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "February 2024 (Leap Year)",
			date:          time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expectedFirst: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expectedLast:  time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFirst := GetMonthFirstDay(tt.date)
			if !gotFirst.Equal(tt.expectedFirst) {
				t.Errorf("GetMonthFirstDay() = %v, want %v", gotFirst, tt.expectedFirst)
			}

			gotLast := GetMonthLastDay(tt.date)
			if !gotLast.Equal(tt.expectedLast) {
				t.Errorf("GetMonthLastDay() = %v, want %v", gotLast, tt.expectedLast)
			}
		})
	}
}

func TestGetDaysInMonth(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    time.Month
		expected int
	}{
		{
			name:     "February 2024 (Leap Year)",
			year:     2024,
			month:    time.February,
			expected: 29,
		},
		{
			name:     "February 2023 (Non-Leap Year)",
			year:     2023,
			month:    time.February,
			expected: 28,
		},
		{
			name:     "April 2024 (30 days)",
			year:     2024,
			month:    time.April,
			expected: 30,
		},
		{
			name:     "December 2024 (31 days)",
			year:     2024,
			month:    time.December,
			expected: 31,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDaysInMonth(tt.year, tt.month); got != tt.expected {
				t.Errorf("GetDaysInMonth() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsBetween(t *testing.T) {
	tests := []struct {
		name      string
		date      time.Time
		startDate time.Time
		endDate   time.Time
		expected  bool
	}{
		{
			name:      "Date within range",
			date:      time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:  true,
		},
		{
			name:      "Date equals start",
			date:      time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:  true,
		},
		{
			name:      "Date equals end",
			date:      time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:  true,
		},
		{
			name:      "Date before range",
			date:      time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:  false,
		},
		{
			name:      "Date after range",
			date:      time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			startDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBetween(tt.date, tt.startDate, tt.endDate); got != tt.expected {
				t.Errorf("IsBetween() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// 基准测试
func BenchmarkWeekdayName(b *testing.B) {
	date := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WeekdayName(date)
	}
}

func BenchmarkGetWorkdayCount(b *testing.B) {
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetWorkdayCount(startDate, endDate)
	}
}

func BenchmarkIsWeekend(b *testing.B) {
	date := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsWeekend(date)
	}
}

func BenchmarkFormatDate(b *testing.B) {
	date := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatDate(date, "YYYY-MM-DD")
	}
}

func BenchmarkGetAge(b *testing.B) {
	birthDate := time.Now().AddDate(-20, 0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetAge(birthDate)
	}
}

func BenchmarkIsLeapYear(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsLeapYear(2024)
	}
}

func BenchmarkGetWeekdayInRange(b *testing.B) {
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetWeekdayInRange(startDate, endDate, 1)
	}
}

func BenchmarkGetDaysInMonth(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDaysInMonth(2024, time.February)
	}
}

// 表组测试
func TestFormatDateTime(t *testing.T) {
	date := time.Date(2024, 3, 18, 15, 30, 45, 0, time.UTC)
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "YYYY-MM-DD HH:mm:ss",
			format:   "YYYY-MM-DD HH:mm:ss",
			expected: "2024-03-18 15:30:45",
		},
		{
			name:     "YYYY/MM/DD HH:mm:ss",
			format:   "YYYY/MM/DD HH:mm:ss",
			expected: "2024/03/18 15:30:45",
		},
		{
			name:     "DD/MM/YYYY HH:mm:ss",
			format:   "DD/MM/YYYY HH:mm:ss",
			expected: "18/03/2024 15:30:45",
		},
		{
			name:     "YYYY年MM月DD日 HH时mm分ss秒",
			format:   "YYYY年MM月DD日 HH时mm分ss秒",
			expected: "2024年03月18日 15时30分45秒",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDateTime(date, tt.format); got != tt.expected {
				t.Errorf("FormatDateTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// 测试覆盖边界情况
func TestAddWorkdays(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		days      int
		expected  time.Time
	}{
		{
			name:      "Add 5 workdays from Monday",
			startDate: time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC),
			days:      5,
			expected:  time.Date(2024, 3, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "Add 5 workdays from Friday",
			startDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			days:      5,
			expected:  time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "Add 0 workdays",
			startDate: time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC),
			days:      0,
			expected:  time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddWorkdays(tt.startDate, tt.days)
			if !got.Equal(tt.expected) {
				t.Errorf("AddWorkdays() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// 测试时区相关功能
func TestConvertTimeZone(t *testing.T) {
	date := time.Date(2024, 3, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "Convert to Asia/Shanghai",
			timezone: "Asia/Shanghai",
			wantErr:  false,
		},
		{
			name:     "Convert to America/New_York",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "Invalid timezone",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertTimeZone(date, tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertTimeZone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Location().String() != tt.timezone {
				t.Errorf("ConvertTimeZone() got timezone = %v, want %v", got.Location().String(), tt.timezone)
			}
		})
	}
}

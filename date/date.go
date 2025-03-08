package date

import (
	"time"
)

// WeekdayName 返回给定日期对应的星期几（1-7，7表示星期日）
func WeekdayName(date time.Time) int {
	weekday := int(date.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday + 1
}

// GetWeekday 返回给定日期对应的星期几（time.Weekday类型）
func GetWeekday(date time.Time) time.Weekday {
	return date.Weekday()
}

// GetWeekdayInt 返回给定日期对应的星期几（整数，0表示星期日，1-6表示星期一至星期六）
func GetWeekdayInt(date time.Time) int {
	return int(date.Weekday())
}

// GetWeekdayInRange 获取指定时间范围内特定星期几的所有日期
// weekday: 0-6，分别代表星期日到星期六
func GetWeekdayInRange(startDate, endDate time.Time, weekday int) []time.Time {
	if weekday < 0 || weekday > 6 {
		return nil
	}

	// 确保开始日期早于结束日期
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// 规范化时间，去除时分秒
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	// 计算第一个符合条件的日期
	firstDate := startDate
	dayDiff := (weekday - int(firstDate.Weekday()) + 7) % 7
	if dayDiff > 0 {
		firstDate = firstDate.AddDate(0, 0, dayDiff)
	}

	// 如果第一个日期已经超出范围，则返回空结果
	if firstDate.After(endDate) {
		return []time.Time{}
	}

	// 计算范围内所有符合条件的日期
	result := []time.Time{firstDate}
	currentDate := firstDate

	for {
		nextDate := currentDate.AddDate(0, 0, 7) // 增加7天
		if nextDate.After(endDate) {
			break
		}
		result = append(result, nextDate)
		currentDate = nextDate
	}

	return result
}

// GetAllWeekdaysInRange 获取指定时间范围内所有星期几的日期，按星期几分组
// 返回一个map，key为0-6（星期日到星期六），value为对应的日期列表
func GetAllWeekdaysInRange(startDate, endDate time.Time) map[int][]time.Time {
	// 确保开始日期早于结束日期
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// 规范化时间，去除时分秒
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	result := make(map[int][]time.Time)

	// 初始化结果map
	for i := 0; i < 7; i++ {
		result[i] = []time.Time{}
	}

	// 遍历日期范围
	currentDate := startDate
	for !currentDate.After(endDate) {
		weekday := int(currentDate.Weekday())
		result[weekday] = append(result[weekday], currentDate)
		currentDate = currentDate.AddDate(0, 0, 1) // 增加一天
	}

	return result
}

// GetWeekdaysInRange 获取指定时间范围内多个星期几的所有日期
// weekdays: []int，每个元素范围为1-7，分别代表星期一至星期日
// 返回一个map，key为输入的星期几（1-7），value为对应的日期列表
func GetWeekdaysInRange(startDate, endDate time.Time, weekdays []int) map[int][]time.Time {
	// 验证输入的星期是否有效
	for _, w := range weekdays {
		if w < 1 || w > 7 {
			return nil
		}
	}

	// 确保开始日期早于结束日期
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// 规范化时间，去除时分秒
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	// 创建结果map
	result := make(map[int][]time.Time)
	for _, w := range weekdays {
		result[w] = []time.Time{}
	}

	// 遍历日期范围
	currentDate := startDate
	for !currentDate.After(endDate) {
		// 将0-6（星期日到星期六）转换为1-7（星期一到星期日）
		weekday := int(currentDate.Weekday())
		if weekday == 0 {
			weekday = 7 // 将星期日从0改为7
		}

		// 如果当前日期的星期在查询列表中，则添加到结果中
		if _, exists := result[weekday]; exists {
			result[weekday] = append(result[weekday], currentDate)
		}

		currentDate = currentDate.AddDate(0, 0, 1) // 增加一天
	}

	return result
}

// GetWorkdayCount 计算两个日期之间的工作日数量（不包含周六和周日）
func GetWorkdayCount(startDate, endDate time.Time) int {
	// 确保开始日期早于结束日期
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// 规范化时间，去除时分秒
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	workdays := 0
	currentDate := startDate

	for !currentDate.After(endDate) {
		weekday := currentDate.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			workdays++
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return workdays
}

// CompareDate 比较两个日期
// 返回值：-1表示date1早于date2，0表示相等，1表示date1晚于date2
func CompareDate(date1, date2 time.Time) int {
	// 规范化时间，去除时分秒
	date1 = time.Date(date1.Year(), date1.Month(), date1.Day(), 0, 0, 0, 0, date1.Location())
	date2 = time.Date(date2.Year(), date2.Month(), date2.Day(), 0, 0, 0, 0, date2.Location())

	if date1.Before(date2) {
		return -1
	} else if date1.After(date2) {
		return 1
	}
	return 0
}

// IsSameDay 判断两个日期是否为同一天
func IsSameDay(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() &&
		date1.Month() == date2.Month() &&
		date1.Day() == date2.Day()
}

// IsWeekend 判断给定日期是否为周末
func IsWeekend(date time.Time) bool {
	weekday := date.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// FormatDate 格式化日期为指定格式
// 支持以下预定义格式：
// "YYYY-MM-DD": 2006-01-02
// "YYYY/MM/DD": 2006/01/02
// "DD/MM/YYYY": 02/01/2006
// "MM/DD/YYYY": 01/02/2006
// "YYYY年MM月DD日": 2006年01月02日
func FormatDate(date time.Time, format string) string {
	switch format {
	case "YYYY-MM-DD":
		return date.Format("2006-01-02")
	case "YYYY/MM/DD":
		return date.Format("2006/01/02")
	case "DD/MM/YYYY":
		return date.Format("02/01/2006")
	case "MM/DD/YYYY":
		return date.Format("01/02/2006")
	case "YYYY年MM月DD日":
		return date.Format("2006年01月02日")
	default:
		return date.Format(format)
	}
}

// ConvertTimeZone 将时间转换为指定时区
func ConvertTimeZone(t time.Time, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return t, err
	}
	return t.In(loc), nil
}

// GetMonthFirstDay 获取指定日期所在月份的第一天
func GetMonthFirstDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

// GetMonthLastDay 获取指定日期所在月份的最后一天
func GetMonthLastDay(date time.Time) time.Time {
	firstDay := GetMonthFirstDay(date)
	lastDay := firstDay.AddDate(0, 1, -1)
	return lastDay
}

// GetQuarterFirstDay 获取指定日期所在季度的第一天
func GetQuarterFirstDay(date time.Time) time.Time {
	month := date.Month()
	quarterFirstMonth := ((month-1)/3)*3 + 1
	return time.Date(date.Year(), quarterFirstMonth, 1, 0, 0, 0, 0, date.Location())
}

// GetQuarterLastDay 获取指定日期所在季度的最后一天
func GetQuarterLastDay(date time.Time) time.Time {
	firstDay := GetQuarterFirstDay(date)
	lastDay := firstDay.AddDate(0, 3, -1)
	return lastDay
}

// GetYearFirstDay 获取指定日期所在年份的第一天
func GetYearFirstDay(date time.Time) time.Time {
	return time.Date(date.Year(), 1, 1, 0, 0, 0, 0, date.Location())
}

// GetYearLastDay 获取指定日期所在年份的最后一天
func GetYearLastDay(date time.Time) time.Time {
	return time.Date(date.Year(), 12, 31, 0, 0, 0, 0, date.Location())
}

// AddWorkdays 增加指定的工作日数量（跳过周末）
func AddWorkdays(date time.Time, days int) time.Time {
	if days == 0 {
		return date
	}

	result := date
	addedWorkdays := 0

	for addedWorkdays < days {
		result = result.AddDate(0, 0, 1)
		if !IsWeekend(result) {
			addedWorkdays++
		}
	}

	return result
}

// SubtractWorkdays 减少指定的工作日数量（跳过周末）
func SubtractWorkdays(date time.Time, days int) time.Time {
	result := date
	remaining := days

	for remaining > 0 {
		result = result.AddDate(0, 0, -1)
		if !IsWeekend(result) {
			remaining--
		}
	}

	return result
}

// GetDateRange 获取两个日期之间的所有日期（包含开始和结束日期）
func GetDateRange(startDate, endDate time.Time) []time.Time {
	// 确保开始日期早于结束日期
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// 规范化时间，去除时分秒
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var dates []time.Time
	currentDate := startDate

	for !currentDate.After(endDate) {
		dates = append(dates, currentDate)
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return dates
}

// FormatDateTime 格式化日期时间为指定格式
// 支持以下预定义格式：
// "YYYY-MM-DD HH:mm:ss": 2006-01-02 15:04:05
// "YYYY/MM/DD HH:mm:ss": 2006/01-02 15:04:05
// "DD/MM/YYYY HH:mm:ss": 02/01/2006 15:04:05
// "MM/DD/YYYY HH:mm:ss": 01/02/2006 15:04:05
// "YYYY年MM月DD日 HH时mm分ss秒": 2006年01月02日 15时04分05秒
func FormatDateTime(date time.Time, format string) string {
	switch format {
	case "YYYY-MM-DD HH:mm:ss":
		return date.Format("2006-01-02 15:04:05")
	case "YYYY/MM/DD HH:mm:ss":
		return date.Format("2006/01/02 15:04:05")
	case "DD/MM/YYYY HH:mm:ss":
		return date.Format("02/01/2006 15:04:05")
	case "MM/DD/YYYY HH:mm:ss":
		return date.Format("01/02/2006 15:04:05")
	case "YYYY年MM月DD日 HH时mm分ss秒":
		return date.Format("2006年01月02日 15时04分05秒")
	default:
		return date.Format(format)
	}
}

// GetAge 根据出生日期计算年龄
func GetAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()

	// 检查是否已经过了今年的生日
	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		years--
	}

	return years
}

// IsLeapYear 判断是否为闰年
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// GetDaysInMonth 获取指定年月的天数
func GetDaysInMonth(year int, month time.Month) int {
	if month == time.February {
		if IsLeapYear(year) {
			return 29
		}
		return 28
	}

	if month == time.April || month == time.June ||
		month == time.September || month == time.November {
		return 30
	}

	return 31
}

// GetDayOfYear 获取指定日期是当年的第几天
func GetDayOfYear(date time.Time) int {
	return date.YearDay()
}

// GetWeekOfYear 获取指定日期是当年的第几周
func GetWeekOfYear(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

// IsBetween 判断日期是否在指定范围内（包含边界）
func IsBetween(date, startDate, endDate time.Time) bool {
	// 规范化时间，去除时分秒
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	return !date.Before(startDate) && !date.After(endDate)
}

// わざとコンパイルできないコードである。
package main

import (
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

func placeBooking(
	member ids.MemberID, lesson ids.LessonID,
	fee domain.Money, slot domain.TimeSlot,
) {
}

func main() {
	memberID := ids.NewID[ids.MemberMarker]()
	lessonID := ids.NewID[ids.LessonMarker]()
	var slot domain.TimeSlot
	placeBooking(lessonID, memberID, -3000, slot)
}

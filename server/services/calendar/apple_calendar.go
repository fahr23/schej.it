package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/jonyTF/go-webdav"
	"github.com/jonyTF/go-webdav/caldav"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/models"
	"schej.it/server/utils"
)

type AppleCalendar struct {
	models.AppleCalendarAuth
}

func (calendar *AppleCalendar) GetCalendarList() (map[string]models.SubCalendar, error) {
	webdavClient, caldavClient, err := calendar.getClients()
	if err != nil {
		return nil, err
	}

	principal, err := webdavClient.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		return nil, err
	}

	calendarHomeSet, err := caldavClient.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		return nil, err
	}

	calendars, err := caldavClient.FindCalendars(context.Background(), calendarHomeSet)
	if err != nil {
		return nil, err
	}

	// Only include calendars that support VEVENT
	filteredCalendars := make(map[string]models.SubCalendar)
	for _, calendar := range calendars {
		for _, supportedComponent := range calendar.SupportedComponentSet {
			if supportedComponent == "VEVENT" {
				filteredCalendars[calendar.Path] = models.SubCalendar{
					Name:    calendar.Name,
					Enabled: utils.TruePtr(),
				}
				break
			}
		}
	}

	return filteredCalendars, nil
}

func (calendar *AppleCalendar) GetCalendarEvents(calendarId string, timeMin time.Time, timeMax time.Time) ([]models.CalendarEvent, error) {
	_, caldavClient, err := calendar.getClients()
	if err != nil {
		return nil, err
	}

	// Get events
	events, err := caldavClient.QueryCalendar(context.Background(), calendarId, &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name: "VEVENT",
				Props: []string{
					"SUMMARY",
					"UID",
					"DTSTART",
					"DTEND",
					"DURATION",
				},
			}},
			Expand: &caldav.CalendarExpandRequest{
				Start: timeMin,
				End:   timeMax,
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: timeMin,
				End:   timeMax,
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	var filteredEvents []models.CalendarEvent
	for _, event := range events {
		startTimeString := event.Data.Children[0].Props["DTSTART"][0].Value
		allDay := false

		var startTime time.Time
		var endTime time.Time

		if !strings.Contains(startTimeString, "T") {
			// Handle all day events
			startTime, _ = time.Parse("20060102", startTimeString)
			endTime, _ = time.Parse("20060102", event.Data.Children[0].Props["DTEND"][0].Value)
			allDay = true
		} else {
			// Handle normal events
			startTime, err = parseTimeWithTZ(event.Data.Children[0].Props.Get("DTSTART"))
			if err != nil {
				continue
			}
			endTime, err = parseTimeWithTZ(event.Data.Children[0].Props.Get("DTEND"))
			if err != nil {
				continue
			}
		}

		filteredEvents = append(filteredEvents, models.CalendarEvent{
			Id:         event.Data.Children[0].Props.Get("UID").Value,
			CalendarId: calendarId,
			Summary:    event.Data.Children[0].Props.Get("SUMMARY").Value,
			StartDate:  primitive.NewDateTimeFromTime(startTime),
			EndDate:    primitive.NewDateTimeFromTime(endTime),
			AllDay:     allDay,
		})
	}

	return filteredEvents, nil
}

func (calendar *AppleCalendar) getClients() (*webdav.Client, *caldav.Client, error) {
	decryptedPassword, err := utils.Decrypt(calendar.Password)
	if err != nil {
		return nil, nil, err
	}

	httpClient := webdav.HTTPClientWithBasicAuth(nil, calendar.Email, decryptedPassword)

	webdavClient, err := webdav.NewClient(httpClient, "https://caldav.icloud.com")
	if err != nil {
		return nil, nil, err
	}

	caldavClient, err := caldav.NewClient(httpClient, "https://caldav.icloud.com")
	if err != nil {
		return nil, nil, err
	}

	return webdavClient, caldavClient, nil
}

func parseTimeWithTZ(prop *ical.Prop) (time.Time, error) {
	timeStr := prop.Value
	tzID := prop.Params.Get("TZID")

	var t time.Time
	var err error

	if tzID != "" {
		loc, err := time.LoadLocation(tzID)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timezone: %v", err)
		}
		//lint:ignore SA4006 err is in fact used later in the code
		t, err = time.ParseInLocation("20060102T150405", timeStr, loc)
	} else {
		t, err = time.Parse("20060102T150405Z", timeStr)
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse time: %v", err)
	}

	return t, nil
}

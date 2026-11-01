package intermodal

import "time"

const (
	chassis   = ChassisNumber("CHAS-0001")
	container = ContainerNumber("TGHU 872349 7")

	// Example stop locations for a Chicago export live-load. All three are real
	// Chicago-area drayage facilities; coordinates are approximate to the street
	// address. time.Time and struct values can't be Go consts, so the
	// appointment windows are kept here as RFC3339 strings (America/Chicago,
	// CST = -06:00) and parsed in liveLoadRunPlan.

	// Chassis pool — ITS ConGlobal / ConGlobal Industries, Forest View.
	poolName    = "ConGlobal Forest View"
	poolLine1   = "6401 W 46th St"
	poolCity    = "Forest View"
	poolRegion  = "IL"
	poolPostal  = "60402"
	poolCountry = "US"
	poolLat     = 41.8120
	poolLng     = -87.7878
	poolCode    = "CGI-FV"

	// Rail ramp — BNSF Corwith Intermodal Facility, Chicago (Brighton Park).
	rampName    = "BNSF Corwith Intermodal Facility"
	rampLine1   = "3526 W 43rd St"
	rampCity    = "Chicago"
	rampRegion  = "IL"
	rampPostal  = "60632"
	rampCountry = "US"
	rampLat     = 41.8147
	rampLng     = -87.7127
	rampCode    = "USCHIBNFB"

	// Customer facility — the "South Side Soapbox" (Method) plant, Pullman.
	shipperName    = "South Side Soapbox (Method), Pullman"
	shipperLine1   = "720 E 111th St"
	shipperCity    = "Chicago"
	shipperRegion  = "IL"
	shipperPostal  = "60628"
	shipperCountry = "US"
	shipperLat     = 41.6922
	shipperLng     = -87.6100
	shipperCode    = "METHOD-PULLMAN"

	// Appointment windows across one export day, in route order.
	poolOutWindowStart = "2026-11-02T06:00:00-06:00" // get chassis
	poolOutWindowEnd   = "2026-11-02T06:30:00-06:00"
	poolOutWindowRef   = "Pool release GOLD-4471"

	rampOutWindowStart = "2026-11-02T07:00:00-06:00" // get empty container
	rampOutWindowEnd   = "2026-11-02T08:00:00-06:00"
	rampOutWindowRef   = "BNSF empty release RSVN-8842"

	shipperWindowStart = "2026-11-02T09:00:00-06:00" // live load at shipper
	shipperWindowEnd   = "2026-11-02T12:00:00-06:00"
	shipperWindowRef   = "Shipper live-load appt APPT-55120"

	rampInWindowStart = "2026-11-02T13:30:00-06:00" // ingate loaded for export
	rampInWindowEnd   = "2026-11-02T15:00:00-06:00"
	rampInWindowRef   = "BNSF ingate RSVN-8842-IN"

	poolInWindowStart = "2026-11-02T15:30:00-06:00" // return chassis
	poolInWindowEnd   = "2026-11-02T16:00:00-06:00"
	poolInWindowRef   = "Pool return GOLD-4471-R"
)

func liveLoadRunPlan() RunPlan {
	mustTime := func(s string) time.Time {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			panic(err)
		}
		return t
	}

	// Physical locations, built once and reused where the route revisits them
	// (chassis pool on the first and last stop, rail ramp on the empty-out and
	// loaded-in stops) so a revisit keeps the same LocationID.
	pool := LocationRef{
		ID:      NewLocationID(),
		Kind:    LocationChassisPool,
		Name:    poolName,
		Address: Address{Line1: poolLine1, City: poolCity, Region: poolRegion, PostalCode: poolPostal, Country: poolCountry},
		Geo:     Coordinates{Lat: poolLat, Lng: poolLng},
		Code:    poolCode,
	}
	ramp := LocationRef{
		ID:      NewLocationID(),
		Kind:    LocationRailRamp,
		Name:    rampName,
		Address: Address{Line1: rampLine1, City: rampCity, Region: rampRegion, PostalCode: rampPostal, Country: rampCountry},
		Geo:     Coordinates{Lat: rampLat, Lng: rampLng},
		Code:    rampCode,
	}
	shipper := LocationRef{
		ID:      NewLocationID(),
		Kind:    LocationCustomerFacility,
		Name:    shipperName,
		Address: Address{Line1: shipperLine1, City: shipperCity, Region: shipperRegion, PostalCode: shipperPostal, Country: shipperCountry},
		Geo:     Coordinates{Lat: shipperLat, Lng: shipperLng},
		Code:    shipperCode,
	}

	stops := []Stop{
		{
			id:       NewStopID(),
			status:   StopPending,
			location: pool,
			window:   AppointmentWindow{Start: mustTime(poolOutWindowStart), End: mustTime(poolOutWindowEnd), Reference: poolOutWindowRef},
			tasks: []Task{
				{
					id:   NewTaskID(),
					kind: TaskPickUpChassis,
					subject: ChassisSubject{
						chassis: ChassisRef{Number: chassis},
					},
					status: TaskPlanned,
				},
			},
		},
		{
			id:       NewStopID(),
			status:   StopPending,
			location: ramp,
			window:   AppointmentWindow{Start: mustTime(rampOutWindowStart), End: mustTime(rampOutWindowEnd), Reference: rampOutWindowRef},
			tasks: []Task{
				{
					id:   NewTaskID(),
					kind: TaskPickUpContainer,
					subject: ContainerSubject{
						container: ContainerRef{Number: container, State: Empty},
					},
					status: TaskPlanned,
				},
			},
		},
		{
			id:       NewStopID(),
			status:   StopPending,
			location: shipper,
			window:   AppointmentWindow{Start: mustTime(shipperWindowStart), End: mustTime(shipperWindowEnd), Reference: shipperWindowRef},
			tasks: []Task{
				{
					id:   NewTaskID(),
					kind: TaskDropOffContainer,
					subject: ContainerSubject{
						container: ContainerRef{Number: container, State: Empty},
					},
					status: TaskPlanned,
				},
				{
					id:   NewTaskID(),
					kind: TaskDropOffChassis,
					subject: ChassisSubject{
						chassis: ChassisRef{Number: chassis},
					},
					status: TaskPlanned,
				},
				{
					id:   NewTaskID(),
					kind: TaskPickUpContainer,
					subject: ContainerSubject{
						container: ContainerRef{Number: container, State: Loaded},
					},
					status: TaskPlanned,
				},
				{
					id:   NewTaskID(),
					kind: TaskPickUpChassis,
					subject: ChassisSubject{
						chassis: ChassisRef{Number: chassis},
					},
					status: TaskPlanned,
				},
			},
		},
		{
			id:       NewStopID(),
			status:   StopPending,
			location: ramp,
			window:   AppointmentWindow{Start: mustTime(rampInWindowStart), End: mustTime(rampInWindowEnd), Reference: rampInWindowRef},
			tasks: []Task{
				{
					id:   NewTaskID(),
					kind: TaskDropOffContainer,
					subject: ContainerSubject{
						container: ContainerRef{Number: container, State: Loaded},
					},
					status: TaskPlanned,
				},
			},
		},
		{
			id:       NewStopID(),
			status:   StopPending,
			location: pool,
			window:   AppointmentWindow{Start: mustTime(poolInWindowStart), End: mustTime(poolInWindowEnd), Reference: poolInWindowRef},
			tasks: []Task{
				{
					id:   NewTaskID(),
					kind: TaskDropOffChassis,
					subject: ChassisSubject{
						chassis: ChassisRef{Number: chassis},
					},
					status: TaskPlanned,
				},
			},
		},
	}
	return RunPlan{Stops: stops}
}

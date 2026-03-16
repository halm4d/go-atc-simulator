package render

const (
	// Font metrics (Ebiten debug font)
	CharWidth  = 6
	CharHeight = 12
	LineHeight = 16

	// Aircraft symbol geometry
	AircraftTipLen         = 9.0
	AircraftTipLenSelected = 11.0
	AircraftWingLen         = 6.0
	AircraftWingLenSelected = 8.0
	WingSpreadDeg          = 140.0
	MaxVelocityVectorPx    = 80.0

	// Data tag layout
	DataTagWidth  = 136
	DataTagHeight = 54
	DataTagRow2Y  = 13 // Y offset for altitude/speed row
	DataTagRow3Y  = 29 // Y offset for heading/route row

	// Waypoint rendering
	WaypointDeltaSize  = 4.0
	WaypointDropRadius = 15.0 // pixel radius for drag-drop detection

	// Ground aircraft
	GroundAircraftSize         = 6.0
	GroundAircraftSizeSelected = 9.0

	// Range rings
	RangeRingSpacingNm = 10.0
	MaxRangeRings      = 6

	// UI panels
	TopPanelHeight    = 48.0
	BottomPanelHeight = 22.0

	// Approach cone
	ApproachConeDistanceNm = 15.0

	// Conflict rendering
	ConflictCircleBaseRadius = 16.0

	// Trail rendering
	TrailDotMaxRadius = 3.0
	TrailDotMinRadius = 1.0

	// Wind rose
	WindRoseRadius = 13.0
)

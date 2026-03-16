package game

const (
	// Window & UI
	DefaultScreenWidth  = 1280
	DefaultScreenHeight = 720
	DefaultZoomLevel    = 8.0
	CommandHistorySize  = 20
	MaxCommandInputLen  = 10

	// Spawning
	SpawnIntervalArrival    = 90.0  // seconds between arrival spawns
	SpawnIntervalDeparture  = 120.0 // seconds between departure spawns
	MinSpawnIntervalArrival = 30.0  // minimum arrival interval at max difficulty
	MinSpawnIntervalDep     = 60.0  // minimum departure interval at max difficulty
	MaxPendingDepartures    = 3
	SpawnSafetyDistanceNm   = 8.0   // minimum horizontal separation for spawning
	SpawnSafetyAltitudeFt   = 2000.0 // minimum vertical separation for spawning
	STARSpawnOffsetMin      = 3.0
	STARSpawnOffsetMax      = 5.0
	RandomSpawnDistMin      = 28.0
	RandomSpawnDistMax      = 35.0
	LandingsPerDifficulty   = 5

	// ILS & Approach
	ILSCaptureDistanceNm        = 15.0
	ApproachHeadingToleranceDeg = 30.0
	GlideSlopeGradientFtPerNm   = 318.0  // 3° ILS ≈ 318 ft/nm
	GlideSlopeAltToleranceFt    = 2000.0
	AutoDescendDistanceNm       = 15.0
	AutoDescendTargetAltFt      = 3000.0

	// Scoring
	InitialScore            = 100
	ScoreLanding            = 50
	ScoreDeparture          = 25
	ConflictPenalty         = 1
	CriticalConflictPenalty = 2

	// Aircraft exit
	ExitRadiusNm = 50.0

	// Wind
	WindSpeedMin          = 2.0
	WindSpeedMax          = 35.0
	WindChangeIntervalSec = 30.0
	WindDirChangeDeg      = 5.0  // ±degrees per interval
	WindSpeedChangeKts    = 2.0  // ±knots per interval
	InitialWindSpeedMin   = 5.0
	InitialWindSpeedMax   = 25.0

	// Drag-to-waypoint
	DragThresholdPxSq = 25 // squared pixel distance (5px threshold)

	// Difficulty scaling per level (subtracted from base interval)
	DifficultyIntervalStep = 10.0

	// Airport menu click radius
	AirportClickRadius = 30.0
)

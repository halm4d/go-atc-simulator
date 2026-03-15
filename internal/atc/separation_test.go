package atc

import (
	"atc-sim/internal/aircraft"
	"testing"
)

func makeAircraft(x, y, alt float64) *aircraft.Aircraft {
	a := aircraft.NewAircraft("TST001", "B738", x, y, alt, 0, 250)
	a.Phase = aircraft.PhaseArrival
	return a
}

func TestCheckSeparation_NoConflict_FarApart(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(20, 0, 10000) // 20nm apart horizontally

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
}

func TestCheckSeparation_NoConflict_VerticalSeparation(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(1, 0, 12000) // 1nm apart but 2000ft vertical

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts due to vertical separation, got %d", len(conflicts))
	}
}

func TestCheckSeparation_Warning(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(4, 0, 10500) // 4nm horiz (<5), 500ft vert (<1000) → WARNING

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Severity != "WARNING" {
		t.Errorf("expected WARNING severity, got %s", conflicts[0].Severity)
	}
}

func TestCheckSeparation_Critical(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(1, 0, 10200) // 1nm horiz (<3), 200ft vert (<500) → CRITICAL

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Severity != "CRITICAL" {
		t.Errorf("expected CRITICAL severity, got %s", conflicts[0].Severity)
	}
}

func TestCheckSeparation_GroundAircraftExcluded(t *testing.T) {
	a1 := makeAircraft(0, 0, 495)
	a1.Phase = aircraft.PhaseHoldingShort
	a2 := makeAircraft(0.1, 0, 495)
	a2.Phase = aircraft.PhaseHoldingShort

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 0 {
		t.Errorf("ground aircraft should be excluded from separation checks, got %d conflicts", len(conflicts))
	}
}

func TestCheckSeparation_DistanceBoundary_ExactlyAtMinimum(t *testing.T) {
	// Exactly at 5nm horizontal and 1000ft vertical — should be clean (not <)
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(5, 0, 11000)

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 0 {
		t.Errorf("aircraft exactly at separation minimum should not be in conflict, got %d", len(conflicts))
	}
}

func TestCheckSeparation_MultipleConflicts(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(1, 0, 10100) // conflicts with a1
	a3 := makeAircraft(0.5, 0, 10050) // conflicts with both

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2, a3})
	if len(conflicts) < 2 {
		t.Errorf("expected at least 2 conflicts, got %d", len(conflicts))
	}
}

func TestCheckSeparation_ConflictFields(t *testing.T) {
	a1 := makeAircraft(0, 0, 10000)
	a2 := makeAircraft(2, 0, 10300)

	conflicts := CheckSeparation([]*aircraft.Aircraft{a1, a2})
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	c := conflicts[0]
	if c.Aircraft1 == nil || c.Aircraft2 == nil {
		t.Error("conflict aircraft pointers should not be nil")
	}
	if c.HorizontalDistance <= 0 {
		t.Error("horizontal distance should be positive")
	}
	if c.VerticalDistance < 0 {
		t.Error("vertical distance should be non-negative")
	}
}

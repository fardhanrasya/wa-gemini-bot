package mining

import (
	"testing"
	"time"
)

func TestCalculateRigStatus(t *testing.T) {
	s := &MiningService{} // Service kosongan karena calculateRigStatus hanya butuh properti Tiers statis

	// 1. Uji Rig Basic Baru (100% Efisiensi, 0 chips mined)
	r1 := Rig{
		ID:            1,
		JID:           "test_user@g.us",
		Tier:          "basic",
		Efficiency:    1.0,
		ChipsMined:    0,
		MaxDurability: 3000,
		LastFuelTime:  time.Now().Add(-2 * time.Hour), // 2 jam yang lalu
	}

	status1 := s.calculateRigStatus(r1)
	if status1.IsBroken {
		t.Errorf("Rig baru seharusnya tidak rusak")
	}
	// Speeds: basic = 10 chips/hour. 2 hours * 10 * 1.0 = 20 chips
	expectedPending := 20
	if status1.PendingChips != expectedPending {
		t.Errorf("Ekspektasi pending chips %d, didapat %d", expectedPending, status1.PendingChips)
	}
	
	// Fuel percent should be around (22 / 24) * 100 = 91.6%
	expectedFuelMin := 91.0
	expectedFuelMax := 92.5
	if status1.FuelPercent < expectedFuelMin || status1.FuelPercent > expectedFuelMax {
		t.Errorf("Ekspektasi fuel percent di antara %.1f%% - %.1f%%, didapat %.1f%%", expectedFuelMin, expectedFuelMax, status1.FuelPercent)
	}

	// 2. Uji Batas Kapasitas Bahan Bakar (Capped at 24 hours)
	r2 := Rig{
		ID:            2,
		JID:           "test_user@g.us",
		Tier:          "basic",
		Efficiency:    1.0,
		ChipsMined:    0,
		MaxDurability: 3000,
		LastFuelTime:  time.Now().Add(-30 * time.Hour), // 30 jam yang lalu (habis bensin)
	}

	status2 := s.calculateRigStatus(r2)
	// Harus dicap pada 24 jam. 24 hours * 10 * 1.0 = 240 chips
	expectedCapped := 240
	if status2.PendingChips != expectedCapped {
		t.Errorf("Ekspektasi pending chips dicap pada %d (24 jam), didapat %d", expectedCapped, status2.PendingChips)
	}
	if status2.FuelPercent != 0.0 {
		t.Errorf("Bahan bakar rig seharusnya 0%% setelah 30 jam, didapat %.1f%%", status2.FuelPercent)
	}

	// 3. Uji Batas Durabilitas (Keausan Rig)
	r3 := Rig{
		ID:            3,
		JID:           "test_user@g.us",
		Tier:          "basic",
		Efficiency:    1.0,
		ChipsMined:    2950, // Tersisa 50 chip sebelum rusak (3000 max)
		MaxDurability: 3000,
		LastFuelTime:  time.Now().Add(-10 * time.Hour), // 10 jam * 10 = 100 chip teoritis
	}

	status3 := s.calculateRigStatus(r3)
	// Pending chips harus dicap di sisa durability (50), bukan 100!
	expectedDurabilityCap := 50
	if status3.PendingChips != expectedDurabilityCap {
		t.Errorf("Ekspektasi pending chips dicap ke sisa durabilitas %d, didapat %d", expectedDurabilityCap, status3.PendingChips)
	}

	// 4. Uji Rig Rusak Total (Fully Expired)
	r4 := Rig{
		ID:            4,
		JID:           "test_user@g.us",
		Tier:          "basic",
		Efficiency:    1.0,
		ChipsMined:    3000, // Sudah habis total
		MaxDurability: 3000,
		LastFuelTime:  time.Now().Add(-2 * time.Hour),
	}

	status4 := s.calculateRigStatus(r4)
	if !status4.IsBroken {
		t.Errorf("Rig seharusnya terdeteksi rusak (Broken)")
	}
	if status4.PendingChips != 0 {
		t.Errorf("Rig rusak seharusnya menghasilkan 0 pending chips, didapat %d", status4.PendingChips)
	}

	// 5. Uji Penurunan Efisiensi (Dampener)
	r5 := Rig{
		ID:            5,
		JID:           "test_user@g.us",
		Tier:          "quantum", // base speed 100/jam
		Efficiency:    0.50,      // Efisiensi unit ke-3 = 50%
		ChipsMined:    0,
		MaxDurability: 50000,
		LastFuelTime:  time.Now().Add(-2 * time.Hour), // 2 jam * 100 * 0.50 = 100 chip
	}

	status5 := s.calculateRigStatus(r5)
	expectedEffSpeed := 50.0 // 100 * 0.5
	if status5.EffectiveSpeed != expectedEffSpeed {
		t.Errorf("Ekspektasi kecepatan efektif %.1f, didapat %.1f", expectedEffSpeed, status5.EffectiveSpeed)
	}
	expectedEffPending := 100
	if status5.PendingChips != expectedEffPending {
		t.Errorf("Ekspektasi pending chips %.d, didapat %d", expectedEffPending, status5.PendingChips)
	}
}

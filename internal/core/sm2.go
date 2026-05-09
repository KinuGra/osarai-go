package core

import "time"

// SM2Params は SM-2 アルゴリズムの現在のパラメータ。
type SM2Params struct {
	EaseFactor  float64
	Interval    int
	Repetitions int
}

// SM2Result は SM-2 アルゴリズムの更新後パラメータ。
type SM2Result struct {
	EaseFactor   float64
	Interval     int
	Repetitions  int
	NextReviewAt time.Time
}

// Calculate は自己評価 rating に基づいて SM-2 パラメータを更新する。
//
// rating: "again" | "hard" | "good" | "easy"
//
// again: repetitions=0, interval=1 にリセット、ease_factor 微減
// hard:  repetitions=0, interval=1 にリセット、ease_factor 微減（again より緩やか）
// good:  SM-2 標準更新
// easy:  SM-2 標準更新（間隔を 1.3 倍で延伸）
func Calculate(params SM2Params, rating string) SM2Result {
	ef := params.EaseFactor
	interval := params.Interval
	reps := params.Repetitions

	switch rating {
	case "again":
		reps = 0
		interval = 1
		ef -= 0.20
	case "hard":
		reps = 0
		interval = 1
		ef -= 0.15
	case "good":
		reps++
		switch reps {
		case 1:
			interval = 1
		case 2:
			interval = 6
		default:
			interval = int(float64(interval) * ef)
		}
	case "easy":
		reps++
		switch reps {
		case 1:
			interval = 4
		case 2:
			interval = 8
		default:
			interval = int(float64(interval) * ef * 1.3)
		}
	}

	if ef < 1.3 {
		ef = 1.3
	}
	if interval < 1 {
		interval = 1
	}

	return SM2Result{
		EaseFactor:   ef,
		Interval:     interval,
		Repetitions:  reps,
		NextReviewAt: time.Now().AddDate(0, 0, interval),
	}
}

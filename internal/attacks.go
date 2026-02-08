package internal




var (

	knightAttacks [91][]int

	kingAttacks [91][]int

	pawnAttacksWhite [91][]int

	pawnAttacksBlack [91][]int



	rookRays [91][6][]int

	bishopRays [91][6][]int


	kingZones [91][]int

	pawnShieldSquaresWhite [91][]int

	pawnShieldSquaresBlack [91][]int


	bishopAdjacent [91][]int
)




func initAttackTables() {

	for idx := 0; idx < 91; idx++ {
		pos := IndexToPosition(idx)
		if !pos.InBoard() {
			continue
		}


		var knightTargets []int
		for _, offset := range knightOffsets {
			target := pos.AddVectorsReturn(offset[0], offset[1])
			if target.InBoard() {
				knightTargets = append(knightTargets, PositionToIndex(target))
			}
		}
		knightAttacks[idx] = knightTargets


		var kingTargets []int
		for _, dirs := range kingOffsets {
			target := pos
			for _, d := range dirs {
				target = target.AddReturn(d)
			}
			if target.InBoard() {
				kingTargets = append(kingTargets, PositionToIndex(target))
			}
		}
		kingAttacks[idx] = kingTargets


		var whitePawnTargets []int
		for _, dir := range []Direction{DirUpLeft, DirUpRight} {
			target := pos.AddReturn(dir)
			if target.InBoard() {
				whitePawnTargets = append(whitePawnTargets, PositionToIndex(target))
			}
		}
		pawnAttacksWhite[idx] = whitePawnTargets


		var blackPawnTargets []int
		for _, dir := range []Direction{DirDownLeft, DirDownRight} {
			target := pos.AddReturn(dir)
			if target.InBoard() {
				blackPawnTargets = append(blackPawnTargets, PositionToIndex(target))
			}
		}
		pawnAttacksBlack[idx] = blackPawnTargets


		for dirIdx, dir := range rookDirs {
			var ray []int
			for mag := 1; ; mag++ {
				target := pos.AddVectorReturn(Vector{dir, mag})
				if !target.InBoard() {
					break
				}
				ray = append(ray, PositionToIndex(target))
			}
			rookRays[idx][dirIdx] = ray
		}


		for dirIdx, dir := range bishopDirs {
			var ray []int
			for mag := 1; ; mag++ {
				target := pos.AddVectorsReturn(Vector{dir[0], mag}, Vector{dir[1], mag})
				if !target.InBoard() {
					break
				}
				ray = append(ray, PositionToIndex(target))
			}
			bishopRays[idx][dirIdx] = ray
		}


		zone := []int{idx}
		for _, srcIdx := range kingAttacks[idx] {
			zone = append(zone, srcIdx)
		}
		kingZones[idx] = zone


		var whiteShield []int
		whiteDirs := []Direction{DirUp, DirUpLeft, DirUpRight}
		for _, dir := range whiteDirs {
			for dist := 1; dist <= 2; dist++ {
				shieldPos := pos
				for i := 0; i < dist; i++ {
					shieldPos = shieldPos.AddReturn(dir)
				}
				if shieldPos.InBoard() {
					whiteShield = append(whiteShield, PositionToIndex(shieldPos))
				}
			}
		}
		pawnShieldSquaresWhite[idx] = whiteShield


		var blackShield []int
		blackDirs := []Direction{DirDown, DirDownLeft, DirDownRight}
		for _, dir := range blackDirs {
			for dist := 1; dist <= 2; dist++ {
				shieldPos := pos
				for i := 0; i < dist; i++ {
					shieldPos = shieldPos.AddReturn(dir)
				}
				if shieldPos.InBoard() {
					blackShield = append(blackShield, PositionToIndex(shieldPos))
				}
			}
		}
		pawnShieldSquaresBlack[idx] = blackShield


		var adjDiag []int
		for _, dir := range bishopDirs {
			target := pos.AddVectorsReturn(Vector{dir[0], 1}, Vector{dir[1], 1})
			if target.InBoard() {
				adjDiag = append(adjDiag, PositionToIndex(target))
			}
		}
		bishopAdjacent[idx] = adjDiag
	}
}


func GetKnightAttacks(idx int) []int {
	return knightAttacks[idx]
}


func GetKingAttacks(idx int) []int {
	return kingAttacks[idx]
}


func GetPawnAttacks(idx int, color Color) []int {
	if color == White {
		return pawnAttacksWhite[idx]
	}
	return pawnAttacksBlack[idx]
}

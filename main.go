package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func debug(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
	fmt.Fprintln(os.Stderr)
}

var dist [][]int
var board [][]int
var score [][]int
var width, height int
var myID int
var maxD int
var distMap map[int][]Pos

var items []Item
var players []Player
var bombs []Bomb

var me Player

const (
	cellFloor    = '.'
	cellWall     = 'X'
	cellBoxEmpty = '0'
	cellBoxRange = '1'
	cellBoxPlus  = '2'
	// BOX = 0, 1, 2
)

// entity type
const (
	EntityPlayer = 0
	EntityBomb   = 1
	EntityItem   = 2
)

// Pos ...
type Pos struct {
	X int
	Y int
}

// Pos3 ...
type Pos3 struct {
	X int
	Y int
	Z int
}

// Pos ...
func (p Pos3) Pos() Pos {
	return Pos{p.X, p.Y}
}

func (p Pos) at(z int) Pos3 {
	return Pos3{p.X, p.Y, z}
}

func (p Pos) adjacent(o Pos) bool {
	if p.X == o.X {
		return abs(p.Y-o.Y) == 1
	} else if p.Y == o.Y {
		return abs(p.X-o.X) == 1
	} else {
		return false
	}
}

func (p Pos3) safePathTo(dest Pos3, bombs []Bomb) Pos3 {
	if p == dest {
		return dest
	}

	debug("want to go %v", dest)

	// debug("bfs start")
	path, _ := bfs(p, bombs, items, func(x, y, d, x0, y0 int, bombs []Bomb, items []Item) bool {
		pos := Pos3{x, y, d}
		// debug("bfs: %d,%d,%d,%d,%d", x, y, d, x0, y0)
		if pos == dest {
			return true
		}
		return false
	})

	return path[0]
}

func (p Pos) down(i int) Pos {
	p.Y += i
	return p
}

func (p Pos) up(i int) Pos {
	p.Y -= i
	return p
}

func (p Pos) left(i int) Pos {
	p.X -= i
	return p
}

func (p Pos) right(i int) Pos {
	p.X += i
	return p
}

// Player ...
type Player struct {
	Pos   Pos
	ID    int
	Bombs int
	Range int
}

// Bomb ...
type Bomb struct {
	Pos       Pos
	Owner     int
	CountDown int
	Range     int
}

const (
	itemNothing    = 0
	itemExtraRange = 1
	itemExtraBomb  = 2
)

// Item ...
type Item struct {
	Pos  Pos
	Type int
}

// TODO: 폭탄이 터지면서 board가 바뀌었을 수도 있다.
func canGo(p Pos3, bombs []Bomb) bool {
	if inRange2D(p.X, p.Y, width, height) && board[p.Y][p.X] == cellFloor {
		for _, b := range bombs {
			if b.Pos == p.Pos() {
				return false
			}
			if b.CountDown-1 == p.Z && b.inRange(p.Pos()) {
				return false
			}
		}
		return true
	}
	return false
}

// SetPos is a set of Pos
type SetPos map[Pos]struct{}

func (set SetPos) add(p Pos) {
	set[p] = struct{}{}
}

func (set SetPos) has(p Pos) bool {
	_, ok := set[p]
	return ok
}

func (set SetPos) toSlice() []Pos {
	slice := make([]Pos, 0, len(set))
	for k := range set {
		slice = append(slice, k)
	}
	sort.Slice(slice, func(i, j int) bool {
		if slice[i].X == slice[j].X {
			return slice[i].Y < slice[j].Y
		}
		return slice[i].X < slice[j].X
	})
	return slice
}

// SetPos3 is set of Pos3
type SetPos3 map[Pos3]struct{}

func (set SetPos3) add(p Pos3) {
	set[p] = struct{}{}
}

func (set SetPos3) has(p Pos3) bool {
	_, ok := set[p]
	return ok
}

func (set SetPos3) toSlice() []Pos3 {
	slice := make([]Pos3, 0, len(set))
	for k := range set {
		slice = append(slice, k)
	}
	sort.Slice(slice, func(i, j int) bool {
		if slice[i].X == slice[j].X {
			if slice[i].Y == slice[j].Y {
				return slice[i].Z < slice[j].Z
			}
			return slice[i].Y < slice[j].Y
		}
		return slice[i].X < slice[j].X
	})
	return slice
}

// bfs 는 시간 축(d)을 고려하고,
// d는 항상 증가하기 때문에,
// 어차피 방문한 곳을 또 방문할 일이 없다.
// 즉, 현 상태의 bombs를 보고
// 안전한 경로로 bfs를 진행해보자.
func bfs(pos Pos3, bombs []Bomb, items []Item, visit func(x, y, d, x0, y0 int, bombs []Bomb, items []Item) bool) ([]Pos3, bool) {
	back := map[Pos3]Pos3{}
	getPath := func(next Pos3) []Pos3 {
		sz := next.Z - pos.Z
		path := make([]Pos3, sz)
		sz--
		path[sz] = next
		for pos != back[next] {
			next = back[next]
			sz--
			path[sz] = next
		}
		// debug("%v", path)
		return path
	}

	layer := []Pos3{pos}
	if visit(pos.X, pos.Y, pos.Z, pos.X, pos.Y, bombs, items) {
		return nil, true
	}

	dxs := []int{0, 0, 1, 0, -1}
	dys := []int{0, 1, 0, -1, 0}
	d := pos.Z
	for i := 0; len(layer) > 0 && i <= width; i++ {

		bombs = removeOld(bombs, d)
		// item도 없어져야 하고..
		// 상자도 없어져야 하고..
		// 그러면 아이템도 생겨야 하고..

		var newLayer = SetPos3{}
		for _, p := range layer {
			for k := 0; k < 5; k++ {
				dx := dxs[k]
				dy := dys[k]
				next := Pos3{p.X + dx, p.Y + dy, p.Z + 1}
				if canGo(next, bombs) && !newLayer.has(next) {
					newLayer.add(next)
					back[next] = p
					if visit(next.X, next.Y, next.Z, p.X, p.Y, bombs, items) {
						return getPath(next), true
					}
				}
			}
		}
		layer = newLayer.toSlice()
	}

	return nil, false
}

// // World ...
// type World struct {
// 	board [][]int
// 	bombs []Bomb
// 	items []Item
// 	d     int
// }

// func (w World) bfs(pos Pos, visit func(w World) bool) {
// 	layer := []Pos{pos}
// 	if visit(w) {
// 		return
// 	}
// 	for len(layer) > 0 && d < d0+9 {
// 		d++
// 		var newLayer = SetPos{}
// 		for _, p := range layer {
// 			// 가만히 있는다
// 			if canGo(p.X, p.Y, d, bombs) {
// 				newLayer.add(p)
// 				if visit(p.X, p.Y, d, p.X, p.Y) {
// 					return
// 				}
// 			}

// 			if canGo(p.X, p.Y-1, d, bombs) {
// 				newLayer.add(Pos{p.X, p.Y - 1})
// 				if visit(p.X, p.Y-1, d, p.X, p.Y) {
// 					return
// 				}
// 			}

// 			if canGo(p.X-1, p.Y, d, bombs) {
// 				newLayer.add(Pos{p.X - 1, p.Y})
// 				if visit(p.X-1, p.Y, d, p.X, p.Y) {
// 					return
// 				}
// 			}

// 			if canGo(p.X, p.Y+1, d, bombs) {
// 				newLayer.add(Pos{p.X, p.Y + 1})
// 				if visit(p.X, p.Y+1, d, p.X, p.Y) {
// 					return
// 				}
// 			}

// 			if canGo(p.X+1, p.Y, d, bombs) {
// 				newLayer.add(Pos{p.X + 1, p.Y})
// 				if visit(p.X+1, p.Y, d, p.X, p.Y) {
// 					return
// 				}
// 			}

// 		}
// 		layer = newLayer.toSlice()
// 	}
// }

func debugB(b [][]int) {
	var line []string
	var lines []string
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			line = append(line, fmt.Sprint(b[h][w]))
		}
		lines = append(lines, strings.Join(line, " "))
		line = nil
	}
	debug(strings.Join(lines, "\n"))
}

func debugM(m map[int][]Pos) {
	max := 0
	for k := range m {
		if k > max {
			max = k
		}
	}
	line := []string{}
	lines := []string{}
	for i := 0; i <= max; i++ {
		if s, ok := m[i]; ok {
			for _, p := range s {
				line = append(line, fmt.Sprintf("(%d,%d)", p.X, p.Y))
			}
			lines = append(lines, strings.Join(line, " "))
			line = nil
		}
	}
	debug(strings.Join(lines, "\n"))
}

func isValid(p Pos) bool {
	return p.X >= 0 && p.X < width && p.Y >= 0 && p.Y < height
}

func isBox(p Pos) bool {
	return board[p.Y][p.X] != cellFloor && board[p.Y][p.X] != cellWall
}

// Box ...
type Box struct {
	Pos  Pos
	Type rune
}

func getBox(p Pos) Box {
	return Box{p, rune(board[p.Y][p.X])}
}

func isWall(p Pos) bool {
	return board[p.Y][p.X] == cellWall
}

func isItem(p Pos) bool {
	for _, e := range items {
		if e.Pos == p {
			return true
		}
	}
	return false
}

func isBomb(p Pos) bool {
	for _, e := range bombs {
		if e.Pos == p {
			return true
		}
	}
	return false
}

func getItem(p Pos) Item {
	for _, e := range items {
		if e.Pos == p {
			return e
		}
	}
	panic("item: no item here")
}

func clear(p Pos) {
	board[p.Y][p.X] = cellFloor
}

// 이걸 언제 놓느냐에 따라서도 결과는 달라진다.
// 내 폭탄 옆에 두면, 이것도 함께 터진다.
// 일단 그냥 터진다고 보자.
// 십자 방향으로 r 만큼 터지는데
// 터지는 방향으로 박스나 아이템이 있으면 거기까지만 터진다.
func explode(pos Pos, r int, destroy bool) []interface{} {
	var destroyed []interface{}

	process := func(p Pos) bool {
		if !isValid(p) {
			return true
		}
		if isWall(p) {
			return true
		}
		if isBox(p) {
			if destroy {
				clear(p)
			}
			destroyed = append(destroyed, getBox(p))
			return true
		}
		if isBomb(p) {
			return true
		}
		if isItem(p) {
			if destroy {
				clear(p)
			}
			destroyed = append(destroyed, getItem(p))
			return true
		}
		return false
	}

	// north
	for i := 0; i < r; i++ {
		p := pos.up(i)
		if process(p) {
			break
		}
	}
	// south
	for i := 0; i < r; i++ {
		p := pos.down(i)
		if process(p) {
			break
		}
	}
	// east
	for i := 0; i < r; i++ {
		p := pos.right(i)
		if process(p) {
			break
		}
	}
	// west
	for i := 0; i < r; i++ {
		p := pos.left(i)
		if process(p) {
			break
		}
	}

	return destroyed
}

// StackInt is stack of int
type StackInt struct {
	values []int
}

func (s *StackInt) push(n int) {
	s.values = append(s.values, n)
}

func (s *StackInt) pop() int {
	sz := len(s.values)
	v := s.values[sz-1]
	s.values = s.values[:sz-1]
	return v
}

func (s *StackInt) isEmpty() bool {
	return len(s.values) == 0
}

func copy1D(r []int) []int {
	result := make([]int, len(r))
	copy(result, r)
	return result
}

func copy2D(m [][]int) [][]int {
	result := make([][]int, len(m))
	for i, row := range m {
		result[i] = copy1D(row)
	}
	return result
}

func inRange(n, lo, hi int) bool {
	return n >= lo && n < hi
}

func inRange2D(x, y, w, h int) bool {
	return inRange(x, 0, w) && inRange(y, 0, h)
}

func explode2(bombs []Bomb, board [][]int, p Pos, s SetPos) {
	if s.has(p) {
		return
	}
	s.add(p)

	x, y := p.X, p.Y
	if board[y][x] < 100 { // not a bomb
		return
	}
	r := bombs[board[y][x]-100].Range
	dxs := []int{1, 0, -1, 0}
	dys := []int{0, 1, 0, -1}

	for d := 0; d < 4; d++ {
		dx := dxs[d]
		dy := dys[d]
		x, y = p.X, p.Y
		for i := 1; i < r; i++ {
			x += dx
			y += dy
			if !inRange2D(x, y, width, height) {
				break
			}
			if board[y][x] == cellFloor {
				continue
			} else if board[y][x] == cellWall {
				break
			} else {
				explode2(bombs, board, Pos{x, y}, s)
				break
			}
		}
	}
}

// syncBombs 는 폭탄의 연쇄폭발로 같이 터지는 폭탄들의
// countdown 값을 일치시켜놓는다.
func syncBombs(bombs []Bomb, board [][]int, items []Item) {
	if len(bombs) < 2 {
		return
	}
	// boards와 items는 실제로 터뜨릴 거다. 그래서 복사해둔다.
	board = copy2D(board)
	// items도 board에 미리 놔둔다. 이건 그냥 보통 박스(0)라고 볼 수 있다.
	for _, item := range items {
		board[item.Pos.Y][item.Pos.X] = cellBoxEmpty
	}
	// 폭탄도 미리 놔두자. 이건 100 + index 로..
	for i, b := range bombs {
		board[b.Pos.Y][b.Pos.X] = 100 + i
	}
	// itemBox(1,2) 는 터뜨려도 또 item이 남는다.

	for d := 1; d <= 9; d++ {
		s := SetPos{}
		for i := range bombs {
			p := bombs[i].Pos
			if bombs[i].CountDown == d && board[p.Y][p.X] >= 100 {
				explode2(bombs, board, bombs[i].Pos, s)
			}
		}

		// 터진 것들을 반영한다.
		for _, p := range s.toSlice() {
			// debug("%d: %v", d, p)
			x, y := p.X, p.Y
			switch board[y][x] {
			case cellBoxPlus, cellBoxRange:
				board[y][x] = cellBoxEmpty
			case cellBoxEmpty:
				board[y][x] = cellFloor
			default:
				// sync 맞춘다.
				bombs[board[y][x]-100].CountDown = d
				board[y][x] = cellFloor
			}
		}
	}
	// for i, b := range bombs {
	// 	debug("bombs[%d] = %v", i, b)
	// }
}

func (r game) move(bomb bool, pos Pos) {
	cmd := "MOVE"
	if bomb {
		cmd = "BOMB"
	}
	fmt.Fprintf(r, "%s %d %d\n", cmd, pos.X, pos.Y)
}

func (r game) init() {
	// begin game
	fmt.Fscan(r, &width, &height, &myID)
}

func (r game) round() bool {

	// read status
	board = make([][]int, height)
	for i := 0; i < height; i++ {
		board[i] = make([]int, width)

		var row string
		fmt.Fscan(r, &row)
		debug(row)
		if len(row) != width {
			debug("wrong input. exit.")
			return false
		}

		for w := 0; w < width; w++ {
			board[i][w] = int(row[w])
		}
	}

	var n int
	fmt.Fscan(r, &n)
	debug("%d", n)

	players = nil
	bombs = nil
	items = nil
	for i := 0; i < n; i++ {
		var entityType, owner, x, y, param1, param2 int
		fmt.Fscan(r, &entityType, &owner, &x, &y, &param1, &param2)
		debug("%d %d %d %d %d %d", entityType, owner, x, y, param1, param2)

		p := Pos{x, y}
		switch entityType {
		case EntityPlayer:
			player := Player{Pos: p, ID: owner, Bombs: param1, Range: param2}
			players = append(players, player)
			if owner == myID {
				me = player
			}
		case EntityBomb:
			bombs = append(bombs, Bomb{Pos: p, Owner: owner, CountDown: param1, Range: param2})
		case EntityItem:
			items = append(items, Item{Pos: p, Type: param1})
		}
	}

	// bombs sync
	syncBombs(bombs, board, items)

	// 우선 주변을 둘러보자.
	// 갈수 있는곳..
	// 뭐가 있을까? 적? 아이템? 박스? 폭탄?

	dropBomb := false
	posToGo := me.Pos.at(0)
	origin := me.Pos.at(0)
	found := false

	type bombScore struct {
		pos  Pos3
		n    int
		toGo Pos3
	}

	debug("item?")
	bfs(origin, bombs, items, func(x, y, d, x0, y0 int, bombs []Bomb, items []Item) bool {
		if d > 4 {
			return true
		}
		pos := Pos3{x, y, d}
		for _, e := range items {
			if e.Pos == pos.Pos() {
				debug("found an item %d,%d,%d", x, y, d)
				// 하지만 먹고나서 괜찮을까?
				safe, ok := me.canEscapeFrom(pos, bombs)
				if !ok {
					debug("but, can't escape from there")
					return false
				}
				debug("safepath from %v=  %v", pos, safe)
				posToGo = pos
				found = true
				return true
			}
		}
		return false
	})

	if !found {
		candidates := []bombScore{}

		bfs(origin, bombs, items, func(x, y, d, x0, y0 int, bombs []Bomb, items []Item) bool {
			pos := Pos3{x, y, d}
			ok, safe, n := me.canDropBomb(pos, bombs)
			if ok {
				// debug("bomb at %v with %d boxes", pos, n)
				candidates = append(candidates, bombScore{pos, n, safe})
			}

			return false
		})

		if len(candidates) > 0 {
			best := candidates[0]
			for _, c := range candidates {
				if best.n < c.n {
					best = c
				}
			}
			found = true
			if best.pos == origin {
				posToGo = best.toGo
				dropBomb = true
			} else {
				posToGo = best.pos
			}
			debug("bomb at %v with %d boxes", best.pos, n)

		}
	}

	if !found {
		debug("stay here? is it safe? let's find a safe place")

		// 8턴을 살아남을 곳 찾아보자.
		// 닥터 스트레인지처럼
		// 각 경우를 따져보아야..
		// 난 어디로 갈까?
		// 상대는 어디로 갈까?
		bt(myID, world{0, board, players, bombs, items})

		var bombsInDanger []Bomb
		for _, b := range bombs {
			if b.inRange(me.Pos) {
				bombsInDanger = append(bombsInDanger, b)
			}
		}

		if len(bombsInDanger) > 0 {
			debug("need to escape from bombs")
			bfs(origin, bombs, items, func(x, y, d, x0, y0 int, bombs []Bomb, items []Item) bool {
				safe := true
				for _, b := range bombs {
					if b.inRange(Pos{x, y}) {
						safe = false
						break
					}
				}
				if safe {
					found = true
					posToGo = Pos3{x, y, d}
					return true
				}
				return false
			})
		}
	}

	// game engine just get shorted path
	// but it can be dangerous
	posToGo = origin.safePathTo(posToGo, bombs)
	if !dropBomb && me.Bombs > 0 {
		debug("however, I  have a bomb")
		ok, _, _ := me.canDropBomb(origin, bombs)
		if ok {
			debug("with bomb drop, need to check if I can escape")
			if posToGo.Z == 0 {
				debug("yes can escape from %v, (already figured out)", posToGo)
				dropBomb = true
			} else if _, ok := me.canEscapeFrom(posToGo, me.dropBomb(bombs)); ok {
				debug("yes can escape from %v", posToGo)
				dropBomb = true
			} else {
				debug("can't escape if i put a bomb here. so just moving!")
			}
		} else {
			debug("noway, bomb'll kill me!")
		}
	}

	// 이제 갈 곳을 결정했다. -> 현재는  목표가 1, 나중에 N개를 평가해보자..

	// 다른 플레이어들로 인해 위험에 처한다면
	// 이 목적지를 버려야 한다.

	// 지금의 선택(폭탄/이동)으로 인해
	// 나의 탈출구가 하나뿐이 되는 상황이라면.
	// 그리고 상대가 그 탈출구를 폭탄으로 막을 수 있다면?

	// 1. 당장 다른 플레이어들이 폭탄을 둔다면?
	// 2.
	// 마지막으로 지금 상대방들이 폭탄을 뒀는데도
	// 목적지로 가서 살수 있을까?
	// 살수 없다면 거기로 가지말자.

	if !surviveIfAllBombs(posToGo, dropBomb, bombs) {
		debug("if others put bombs, I may die from %v", posToGo)
		if dropBomb && surviveIfAllBombs(posToGo, false, bombs) {
			debug("if i don't drop bomb, it's okay")
			dropBomb = false
		} else if dropBomb && surviveIfAllBombs(origin, dropBomb, bombs) {
			debug("if can survive from origin with bomb")
			path, _ := me.canEscapeFrom(origin, allBombs(dropBomb, bombs))
			posToGo = path[0]
		} else if dropBomb && surviveIfAllBombs(origin, false, bombs) {
			debug("if can survive from origin without bomb")
			path, _ := me.canEscapeFrom(origin, allBombs(false, bombs))
			posToGo = path[0]
			dropBomb = false
		} else if surviveIfAllBombs(origin, false, bombs) {
			debug("if can survive from origin")
			path, _ := me.canEscapeFrom(origin, allBombs(false, bombs))
			posToGo = path[0]
		} else {
			debug("doomed!")
		}
	}

	r.move(dropBomb, posToGo.Pos())

	// 	// 이때 도망가는 중에도 폭탄을 떨어뜨릴지 고민해보자
	// 	// 일단 도망
	// 	r.move(dropBomb, posToGo)
	// 	return
	// }

	// debug("it's okay, safe here")
	// // 시급하면 바로 도망. 아니면 뭔가 다른 행동을 해도.. (단 조심하면서..)
	// // 폭탄은 연쇄 폭발하고.
	// // 폭발로 인해 새로운 길이 생길수도 있다.
	// // 연쇄 폭발을 이용할 수도 있다. 내 폭탄이 있고 나한테 폭탄이 더 있다면..  range 내에 폭탄을 놓고 대피 하는 식으로 range 를 확대할 수 있다.
	// // 		r.move(dropBomb, posToGo)
	// // 		return

	// // 주위에 아무것도 없다.
	// // 그럼 같힌 상태..
	// // 박스를 터뜨려서 아이템을 획득할 수 있다.
	// // 어디에 폭탄을 놓을까?
	// // 아이템을 얻거나 새로운 길을 만들거나 하는 곳
	// // 그러면서 나를 가두지 않는 곳.

	// // 아니면 이곳 말고 다른 곳.. 어디로 가지?

	// 갈수 없는곳도 봐야 할까? 적과 연결되어있는지.. 얼마나 가로막혀있는지에 따라 전략이 달라질 수 있지 않을까?

	// 갈 수 있는 곳에 bomb 이 있나?
	//   countDown = 0 시점에는 range 바깥에 있어야 한다.
	//   피할 수 있는 곳이 bomb countdown 거리 내에 있나? 그럼 피하자
	// range 바

	return true
}

type world struct {
	t       int
	board   [][]int
	players []Player
	bombs   []Bomb
	items   []Item
}

func (w world) String() string {
	return ""
}

type move struct {
}

func (w world) next() []world {
	return nil
}

func (w world) winner() int {
	return -1
}

func bt(id int, w world) {
	switch winner := w.winner(); winner {
	case -1:
		break
	default:
		return
	}
	// 각 player 의 next move 를 따라가보자.
	for _, w := range w.next() {
		bt(id, w)
	}
}

func allBombs(dropBomb bool, bombs []Bomb) []Bomb {
	for _, p := range players {
		if p.ID == myID {
			if dropBomb {
				bombs = p.dropBomb(bombs)
			}
			continue
		}
		if p.Bombs > 0 {
			bombs = p.dropBomb(bombs)
		}
	}
	return bombs
}

func surviveIfAllBombs(p Pos3, dropBomb bool, bombs []Bomb) bool {
	_, ok := me.canEscapeFrom(p, allBombs(dropBomb, bombs))
	return ok
}

// d0 시간뒤에 pos 에서 탈출할 수 있을까?
// 탈출 가능한 곳을 두어개 찾을 수 있어야 한다.
// 반환값은 가능한 목록??
// 그리고 거기가 다른 플레이어에게 더 가까우면 안된다.
func (p Player) canEscapeFrom(pos Pos3, bombs []Bomb) ([]Pos3, bool) {
	return bfs(pos, bombs, items, func(x, y, d, x0, y0 int, bs []Bomb, is []Item) bool {
		// here := Pos3{x, y, d}
		safe := true
		for _, b := range bs {
			if b.inRange(Pos{x, y}) {
				safe = false
				break
			}
		}

		if safe {
			// debug("seems safe %v", here)
			// for _, o := range players {
			// 	if o.ID == p.ID {
			// 		continue
			// 	}
			// 	// o가 먼저 여기에 도착하면 안된다.
			// 	closer := false
			// 	bfs(o.Pos.at(pos.Z), bombs, items, func(x, y, d, x0, y0 int, bs []Bomb, is []Item) bool {
			// 		if d >= here.Z {
			// 			return true
			// 		}
			// 		if x == here.X && y == here.Y {
			// 			closer = true
			// 			return true
			// 		}
			// 		return false
			// 	})
			// 	if closer {
			// 		// debug("but %d may block me", o.ID)
			// 		return false
			// 	}
			// }
			// debug("safe!")
			return true
		}
		return false
	})
}

func (p Player) dropBomb(bombs []Bomb) []Bomb {
	b := Bomb{
		Pos:       p.Pos,
		Owner:     p.ID,
		Range:     p.Range,
		CountDown: 8 + 1,
	}

	var bombs2 []Bomb
	bombs2 = append(bombs2, bombs...)
	bombs2 = append(bombs2, b)
	syncBombs(bombs2, board, items)
	return bombs2
}

func removeOld(bombs []Bomb, d int) []Bomb {
	var result []Bomb
	for _, b := range bombs {
		if b.CountDown > d {
			result = append(result, b)
		}
	}
	return result
}

// 폭탄을 놓을 수 있나?
// 놓아서 터질 박스는 있나? 죽지않고 피할 장소는?
// 이미 놓여있는 bomb 들도 피해야 한다.
func (p Player) canDropBomb(pos Pos3, bombs []Bomb) (canDrop bool, safePlace Pos3, count int) {
	// bombs 에서 시간 지난거 필터링하자
	var n int
	for _, b := range bombs {
		if b.Owner == p.ID && b.CountDown <= pos.Z {
			n++
		}
	}

	// n만큼 플레이어한테 폭탄이 추가되어야 한다.
	// 놓을 폭탄이 없을수도 ...
	if p.Bombs+n == 0 {
		return
	}

	bombs = removeOld(bombs, pos.Z)

	b := Bomb{
		Pos:       pos.Pos(),
		Owner:     p.ID,
		Range:     p.Range,
		CountDown: 9 + pos.Z,
	}

	bombs2 := make([]Bomb, len(bombs))
	copy(bombs2, bombs)
	bombs2 = append(bombs2, b)
	syncBombs(bombs2, board, items)

	destroyed := explode(b.Pos, b.Range, false)
	for _, d := range destroyed {
		if _, ok := d.(Box); ok {
			count++
		}
	}

	if count == 0 {
		return
	}
	path, canDrop := p.canEscapeFrom(pos, bombs2)
	if canDrop {
		safePlace = path[0]
	}
	return
}

// bomb 터질 범위에 있나?
// from 에 있는 bomb
func (b Bomb) inRange(to Pos) bool {
	// 같은 수직선 상에..
	if b.Pos.X == to.X {
		return abs(b.Pos.Y-to.Y) < b.Range
	} else if b.Pos.Y == to.Y {
		return abs(b.Pos.X-to.X) < b.Range
	} else {
		return false
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

type game struct {
	io.Reader
	io.Writer
}

func main() {
	var r io.Reader
	if len(os.Args) > 1 {
		r, _ = os.Open(os.Args[1])
	} else {
		r = os.Stdin
	}

	g := game{r, os.Stdout}
	g.init()
	for {
		on := g.round()
		if !on {
			break
		}
	}
}

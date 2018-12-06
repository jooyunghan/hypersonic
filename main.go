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
	Floor = '.'
	Wall  = 'X'
	// BOX = 0, 1, 2
)

// entity type
const (
	EntityPlayer = 0
	EntityBomb   = 1
	EntityItem   = 2
)

type Pos struct {
	X int
	Y int
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
func (src Pos) safePathTo(dest Pos, bombs []Bomb) Pos {
	if src == dest {
		return dest
	}

	debug("want to go %v", dest)

	back := make([][]Pos, height)
	for h := 0; h < height; h++ {
		back[h] = make([]Pos, width)
	}

	bfs(src, 0, bombs, func(x, y, d, x0, y0 int) bool {
		pos := Pos{x, y}
		if pos != src {
			back[y][x] = Pos{x0, y0}
		}
		if pos == dest {
			return true
		}
		return false
	})

	// backtrack
	next := dest
	for src != back[next.Y][next.X] {
		next = back[next.Y][next.X]
	}
	return next
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

type Player struct {
	Pos
	ID    int
	Bombs int
	Range int
}

type Bomb struct {
	Pos
	Owner     int
	CountDown int
	Range     int
}

const (
	ItemNothing    = 0
	ItemExtraRange = 1
	ItemExtraBomb  = 2
)

type Item struct {
	Pos
	Type int
}

func canGo(x, y, d int, bombs []Bomb) bool {
	p := Pos{x, y}
	if x >= 0 && y >= 0 && x < width && y < height && board[y][x] == Floor {
		for _, b := range bombs {
			if b.Pos == p {
				return false
			}
			if b.CountDown-1 == d && b.inRange(p) {
				return false
			}
		}
		return true
	}
	return false
}

func bfs(pos Pos, d0 int, bombs []Bomb, visit func(x, y, d, x0, y0 int) bool) {
	INF := width + height + 1

	dist := make([][]int, height)
	for h := 0; h < height; h++ {
		dist[h] = make([]int, width)
		for w := 0; w < width; w++ {
			dist[h][w] = INF
		}
	}
	layer := []Pos{pos}
	d := d0
	dist[pos.Y][pos.X] = d
	if visit(pos.X, pos.Y, d, pos.X, pos.Y) {
		return
	}
	for len(layer) > 0 {
		d++
		var newLayer []Pos
		for _, p := range layer {

			// visit neighbor unvisited
			if canGo(p.X, p.Y-1, d, bombs) && dist[p.Y-1][p.X] == INF {
				dist[p.Y-1][p.X] = d
				newLayer = append(newLayer, Pos{p.X, p.Y - 1})
				if visit(p.X, p.Y-1, d, p.X, p.Y) {
					return
				}
			}
			if canGo(p.X-1, p.Y, d, bombs) && dist[p.Y][p.X-1] == INF {
				dist[p.Y][p.X-1] = d
				newLayer = append(newLayer, Pos{p.X - 1, p.Y})
				if visit(p.X-1, p.Y, d, p.X, p.Y) {
					return
				}
			}
			if canGo(p.X, p.Y+1, d, bombs) && dist[p.Y+1][p.X] == INF {
				dist[p.Y+1][p.X] = d
				newLayer = append(newLayer, Pos{p.X, p.Y + 1})
				if visit(p.X, p.Y+1, d, p.X, p.Y) {
					return
				}
			}
			if canGo(p.X+1, p.Y, d, bombs) && dist[p.Y][p.X+1] == INF {
				dist[p.Y][p.X+1] = d
				newLayer = append(newLayer, Pos{p.X + 1, p.Y})
				if visit(p.X+1, p.Y, d, p.X, p.Y) {
					return
				}
			}

		}
		layer = newLayer
	}
}

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
	for k, _ := range m {
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
	return board[p.Y][p.X] != Floor && board[p.Y][p.X] != Wall
}

type Box struct {
	Pos
	Type int
}

func getBox(p Pos) Box {
	return Box{p, board[p.Y][p.X]}
}

func isWall(p Pos) bool {
	return board[p.Y][p.X] == Wall
}

func isItem(p Pos) bool {
	for _, e := range items {
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
	board[p.Y][p.X] = Floor
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

func syncBombs(bombs []Bomb) {
	stack := []int{}
	size := 0

	push := func(n int) {
		stack = append(stack, n)
		size++
	}
	pop := func() int {
		size--
		value := stack[size]
		stack = stack[0:size]
		return value
	}
	empty := func() bool {
		return size == 0
	}
	if len(bombs) < 2 {
		return
	}
	sort.Slice(bombs, func(i, j int) bool {
		return bombs[i].CountDown < bombs[j].CountDown
	})

	for i := len(bombs) - 1; i >= 0; i-- {
		push(i)
	}
	did := map[int]struct{}{}
	for !empty() {
		i := pop()
		if _, ok := did[i]; ok {
			continue
		}
		// explode
		did[i] = struct{}{}

		for j := 0; j < len(bombs); j++ {
			if _, ok := did[j]; ok {
				continue
			}
			if bombs[i].inRange(bombs[j].Pos) {
				bombs[j].CountDown = bombs[i].CountDown
				push(j)
			}
		}
	}

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

func (r game) round() {

	// read status
	board = make([][]int, height)
	for i := 0; i < height; i++ {
		board[i] = make([]int, width)

		var row string
		fmt.Fscan(r, &row)
		debug(row)

		for w := 0; w < width; w++ {
			board[i][w] = int(row[w])
		}
	}

	var n int
	fmt.Fscan(r, &n)

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
	syncBombs(bombs)

	// 우선 주변을 둘러보자.
	// 갈수 있는곳..
	// 뭐가 있을까? 적? 아이템? 박스? 폭탄?

	dropBomb := false
	posToGo := me.Pos
	found := false

	debug("item?")
	bfs(me.Pos, 0, bombs, func(x, y, d, x0, y0 int) bool {
		pos := Pos{x, y}
		for _, e := range items {
			if e.Pos == pos {
				debug("found an item, go get it")
				// 하지마 여기가 괜찮을까?
				if !me.canEscapeFrom(pos, d, bombs) {
					debug("but, can't escape from there")
					return false
				}

				posToGo = pos
				found = true
				return true
			}
		}
		return false
	})

	if !found && me.Bombs > 0 {
		debug("I have a bomb, box here?")
		ok, safe := me.canDropBomb(me.Pos, 0)
		if ok {
			debug("Yes")
			dropBomb = true
			posToGo = safe
			found = true
		}
	}

	if !found {
		debug("box (not here)?")
		bfs(me.Pos, 0, bombs, func(x, y, d, x0, y0 int) bool {
			pos := Pos{x, y}
			if pos == me.Pos {
				return false
			}
			ok, _ := me.canDropBomb(pos, d)
			if ok {
				debug("found place to put a bomb")
				if !me.canEscapeFrom(pos, d, bombs) {
					debug("but, can't escape from there")
					return false
				}
				posToGo = pos
				found = true
				return true
			}
			return false
		})
	}

	if !found {
		debug("stay here? is it safe? let's find a safe place")
		var bombsInDanger []Bomb
		for _, b := range bombs {
			if b.inRange(me.Pos) {
				bombsInDanger = append(bombsInDanger, b)
			}
		}

		if len(bombsInDanger) > 0 {
			debug("need to escape from bombs")
			bfs(me.Pos, 0, bombs, func(x, y, d, x0, y0 int) bool {
				safe := true
				for _, b := range bombs {
					if b.inRange(Pos{x, y}) {
						safe = false
						break
					}
				}
				if safe {
					found = true
					posToGo = Pos{x, y}
					return true
				}
				return false
			})
		}
	}

	// game engine just get shorted path
	// but it can be dangerous
	posToGo = me.Pos.safePathTo(posToGo, bombs)
	if !dropBomb && me.Bombs > 0 {
		debug("however, I  have a bomb")
		ok, _ := me.canDropBomb(me.Pos, 0)
		if ok {
			debug("with bomb drop, need to check if I can escape")
			if posToGo == me.Pos || me.canEscapeFrom(posToGo, 1, me.dropBomb(bombs)) {
				debug("yes can escape from %v", posToGo)
				dropBomb = true
			} else {
				debug("can't escape if i put a bomb here. so just moving!")
			}
		} else {
			debug("noway, bomb'll kill me!")
		}
	}

	r.move(dropBomb, posToGo)

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

}

// d0 시간뒤에 pos 에서 탈출할 수 있을까?
func (p Player) canEscapeFrom(pos Pos, d0 int, bombs []Bomb) bool {
	found := false
	bfs(pos, d0, bombs, func(x, y, d, x0, y0 int) bool {
		safe := true
		for _, b := range bombs {
			if b.inRange(Pos{x, y}) {
				safe = false
				break
			}
		}
		if safe {
			found = true
			return true
		}
		return false
	})
	return found
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
	syncBombs(bombs2)
	return bombs2
}

// 놓을 수 있나? 놓아서 터질 박스는 있나? 죽지않고 피할 장소는?
// 이미 놓여있는 bomb 들도 피해야 한다.
func (p Player) canDropBomb(pos Pos, d0 int) (canDrop bool, safePlace Pos) {
	b := Bomb{
		Pos:       pos,
		Owner:     p.ID,
		Range:     p.Range,
		CountDown: 8 + 1,
	}

	var bombs2 []Bomb
	bombs2 = append(bombs2, bombs...)
	bombs2 = append(bombs2, b)
	syncBombs(bombs2)

	destroyed := explode(b.Pos, b.Range, false)
	var destroyedBoxes []Box
	for _, d := range destroyed {
		if b, ok := d.(Box); ok {
			destroyedBoxes = append(destroyedBoxes, b)
		}
	}
	if len(destroyedBoxes) == 0 {
		return false, Pos{}
	}
	if !p.canEscapeFrom(pos, d0, bombs) {
		return false, Pos{}
	}

	bfs(pos, d0, bombs2, func(x, y, d, x0, y0 int) bool {
		// 거리가 8 이하고
		// me.Pos 에서 터졌을때 range 에 들어가지 않아야 함.
		visit := Pos{x, y}

		for _, bb := range bombs2 {
			if bb.inRange(visit) {
				return false
			}
		}
		if d <= b.CountDown {
			canDrop = true
			safePlace = visit
			return true
		}
		return false
	})
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
	var r io.Reader = os.Stdin

	g := game{r, os.Stdout}

	g.init()
	for {
		g.round()
	}
}

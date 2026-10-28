package hereticsmoltenblade

import (
	"fmt"

	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

// Follows Weapon_Sword_SerpentTooth (ConfigAbility_Weapon_Sword2) with the
// params of EquipAffix 111435 (Duration, TriggerCD, MinAttackRatio,
// MaxAttackRatio, SpeedForMax):
//   - the wearer using their Elemental Skill while TriggerCD is off attaches
//     Gleam of First Light for 14s and starts the 14s TriggerCD
//   - Gleam of First Light gives ATK +MinAttackRatio as soon as it is
//     attached; every second it is recomputed from the distance moved during
//     that second as Min + (Max-Min) * distance / SpeedForMax, capped at Max
//     from SpeedForMax (7m) on
//   - leaving the field removes Gleam of First Light (the TriggerCD stays)
//
// The distance is what the rotation reports with movement() or an action's
// movement param (event.OnMovement), the way Iansan's burst counts it: the
// meters reported while Gleam of First Light is up are summed and read every
// second. A rotation that reports none keeps the minimum bonus.

const (
	buffDur     = 14 * 60
	cdDur       = 14 * 60
	tickDur     = 60
	speedForMax = 7.0

	buffKey = "heretics-molten-blade-gleam"
	cdKey   = "heretics-molten-blade-cd"
)

type Weapon struct {
	Index int
	src   int
	moved float64
}

func (w *Weapon) SetIndex(idx int) { w.Index = idx }
func (w *Weapon) Init() error      { return nil }

func NewWeapon(c *core.Core, char *character.CharWrapper, p info.WeaponProfile) (info.Weapon, error) {
	w := &Weapon{src: -1}
	r := float64(p.Refine)

	minATK := 0.135 + r*0.045
	maxATK := 0.27 + r*0.09

	m := make([]float64, attributes.EndStatType)

	// every second the bonus is recomputed from the distance moved during it
	var tick func(src int) func()
	tick = func(src int) func() {
		return func() {
			if w.src != src || !char.StatModIsActive(buffKey) {
				return
			}
			m[attributes.ATKP] = minATK + (maxATK-minATK)*min(w.moved/speedForMax, 1)
			w.moved = 0
			char.QueueCharTask(tick(src), tickDur)
		}
	}

	c.Events.Subscribe(event.OnSkill, func(args ...any) {
		if c.Player.Active() != char.Index() {
			return
		}
		if char.StatusIsActive(cdKey) {
			return
		}
		char.AddStatus(cdKey, cdDur, true)
		w.src = c.F
		w.moved = 0
		m[attributes.ATKP] = minATK
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(buffKey, buffDur),
			AffectedStat: attributes.ATKP,
			Amount: func() []float64 {
				return m
			},
		})
		char.QueueCharTask(tick(w.src), tickDur)
	}, fmt.Sprintf("heretics-molten-blade-skill-%v", char.Base.Key.String()))

	// Gleam of First Light is removed when the wearer leaves the field, so
	// all movement while it is up is the wearer's
	c.Events.Subscribe(event.OnMovement, func(args ...any) {
		if !char.StatModIsActive(buffKey) {
			return
		}
		w.moved += args[0].(float64)
	}, fmt.Sprintf("heretics-molten-blade-movement-%v", char.Base.Key.String()))

	c.Events.Subscribe(event.OnCharacterSwap, func(args ...any) {
		prev := args[0].(int)
		if prev != char.Index() {
			return
		}
		char.DeleteStatMod(buffKey)
		w.src = -1
	}, fmt.Sprintf("heretics-molten-blade-swap-%v", char.Base.Key.String()))

	return w, nil
}

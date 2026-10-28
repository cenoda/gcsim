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
// gcsim does not track movement, so the distance moved each second is the
// weapon param "distance" (whole meters, default 0: a stationary wearer keeps
// the minimum bonus). distance=7 or more gives the maximum.

const (
	buffDur     = 14 * 60
	cdDur       = 14 * 60
	tickDur     = 60
	speedForMax = 7.0

	buffKey  = "heretics-molten-blade-gleam"
	cdKey    = "heretics-molten-blade-cd"
	firstKey = "heretics-molten-blade-first-second"
)

type Weapon struct {
	Index int
}

func (w *Weapon) SetIndex(idx int) { w.Index = idx }
func (w *Weapon) Init() error      { return nil }

func NewWeapon(c *core.Core, char *character.CharWrapper, p info.WeaponProfile) (info.Weapon, error) {
	w := &Weapon{}
	r := float64(p.Refine)

	minATK := 0.135 + r*0.045
	maxATK := 0.27 + r*0.09

	distance := min(max(float64(p.Params["distance"]), 0), speedForMax)
	moved := minATK + (maxATK-minATK)*distance/speedForMax

	m := make([]float64, attributes.EndStatType)

	c.Events.Subscribe(event.OnSkill, func(args ...any) {
		if c.Player.Active() != char.Index() {
			return
		}
		if char.StatusIsActive(cdKey) {
			return
		}
		char.AddStatus(cdKey, cdDur, true)
		// the first recompute happens one second in; until then the bonus is
		// the minimum
		char.AddStatus(firstKey, tickDur, true)
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(buffKey, buffDur),
			AffectedStat: attributes.ATKP,
			Amount: func() []float64 {
				if char.StatusIsActive(firstKey) {
					m[attributes.ATKP] = minATK
				} else {
					m[attributes.ATKP] = moved
				}
				return m
			},
		})
	}, fmt.Sprintf("heretics-molten-blade-skill-%v", char.Base.Key.String()))

	c.Events.Subscribe(event.OnCharacterSwap, func(args ...any) {
		prev := args[0].(int)
		if prev != char.Index() {
			return
		}
		char.DeleteStatMod(buffKey)
		char.DeleteStatus(firstKey)
	}, fmt.Sprintf("heretics-molten-blade-swap-%v", char.Base.Key.String()))

	return w, nil
}

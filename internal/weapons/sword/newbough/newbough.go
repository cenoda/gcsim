package newbough

import (
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

// Follows Weapon_Sword_SpikedStake (ConfigAbility_Weapon_Sword2) with the params of
// EquipAffix 111437:
//   - using the Elemental Skill opens a 12s window (refreshed by each use)
//   - during the window, a hit by the wearer on an opponent adds a Verdant stack, once every 1s
//     (one ICD for all hits, max 3 stacks). Every gain refreshes the 6s duration of all stacks,
//     and all of them are lost together when it runs out
//   - per stack: ATK +4% and EM +20, or, while the wearer is in either Radiance state
//     (_IS_IN_STARMODE), ATK +6% and Stellar-Conduct / Stellar Swirl reaction DMG +8% instead.
//     The state is read continuously, so the buff switches as Radiance comes and goes
//
// Everything also works off-field.

const (
	windowDur = 12 * 60
	buffDur   = 6 * 60
	hitICD    = 1 * 60
	maxStacks = 3

	windowKey  = "new-bough-window"
	icdKey     = "new-bough-icd"
	atkKey     = "new-bough-verdant-atk"
	emKey      = "new-bough-verdant-em"
	stellarKey = "new-bough-verdant-stellar"
)

type Weapon struct {
	Index  int
	stacks int
}

func (w *Weapon) SetIndex(idx int) { w.Index = idx }
func (w *Weapon) Init() error      { return nil }

// radiance is _IS_IN_STARMODE: Radiance: Stellar-Conduct inside a Polestar Field or Radiance:
// Stellar Swirl after a Stellar Swirl
func radiance(char *character.CharWrapper) bool {
	return char.StatusIsActive(reactable.PolestarFieldKey) || char.StatusIsActive(reactable.RadianceSwirlKey)
}

func NewWeapon(c *core.Core, char *character.CharWrapper, p info.WeaponProfile) (info.Weapon, error) {
	w := &Weapon{}
	r := float64(p.Refine)

	atk := 0.03 + r*0.01
	atkStar := 0.045 + r*0.015
	em := 15 + r*5
	stellar := 0.06 + r*0.02

	mATK := make([]float64, attributes.EndStatType)
	mEM := make([]float64, attributes.EndStatType)

	c.Events.Subscribe(event.OnSkill, func(args ...any) {
		if c.Player.Active() != char.Index() {
			return
		}
		char.AddStatus(windowKey, windowDur, true)
	}, "new-bough-skill-"+char.Base.Key.String())

	c.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		ae := args[1].(*info.AttackEvent)
		if ae.Info.ActorIndex != char.Index() {
			return
		}
		if !char.StatusIsActive(windowKey) {
			return
		}
		if char.StatusIsActive(icdKey) {
			return
		}
		char.AddStatus(icdKey, hitICD, true)

		if !char.StatModIsActive(atkKey) {
			w.stacks = 0
		}
		w.stacks = min(w.stacks+1, maxStacks)

		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(atkKey, buffDur),
			AffectedStat: attributes.ATKP,
			Amount: func() []float64 {
				if radiance(char) {
					mATK[attributes.ATKP] = atkStar * float64(w.stacks)
				} else {
					mATK[attributes.ATKP] = atk * float64(w.stacks)
				}
				return mATK
			},
		})
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(emKey, buffDur),
			AffectedStat: attributes.EM,
			Amount: func() []float64 {
				if radiance(char) {
					return nil
				}
				mEM[attributes.EM] = em * float64(w.stacks)
				return mEM
			},
		})
		char.AddReactBonusMod(character.ReactBonusMod{
			Base: modifier.NewBaseWithHitlag(stellarKey, buffDur),
			Amount: func(ai info.AttackInfo) float64 {
				if !ai.AttackTag.IsStellar() {
					return 0
				}
				if !radiance(char) {
					return 0
				}
				return stellar * float64(w.stacks)
			},
		})
	}, "new-bough-hit-"+char.Base.Key.String())

	return w, nil
}

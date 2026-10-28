package hymnofthemaelstrom

import (
	"fmt"

	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

// Follows Weapon_Catalyst_Bludnye (ConfigAbility_Weapon_Catalyst2) with the
// params of EquipAffix 114524, mapped by ConfigAffix_Catalyst:
//   - each heal the wearer performs adds a Vatsamonga's Vatic Vintage stack
//     (10s each, max 3; at the cap a heal refreshes every stack)
//   - per stack: wearer Max HP +4% and active member ATK +0.4% per 1000 of the
//     wearer's Max HP over 40000, at most 8% per stack
//   - a Freeze or Stellar Swirl on an opponent by any party member multiplies
//     both by 1.75 for 5s
//
// Healing Bonus comes from the affix's addProps.

const (
	stackDur  = 10 * 60
	maxStacks = 3
	strongDur = 5 * 60
	minHP     = 40000.0
	strongMul = 1.75

	strongKey = "hymn-of-the-maelstrom-strong"
)

type Weapon struct {
	Index   int
	initATK func()
}

func (w *Weapon) SetIndex(idx int) { w.Index = idx }

// the party is complete only once every weapon is constructed
func (w *Weapon) Init() error {
	w.initATK()
	return nil
}

func stackKey(i int) string {
	return fmt.Sprintf("hymn-of-the-maelstrom-stack-%v", i)
}

func NewWeapon(c *core.Core, char *character.CharWrapper, p info.WeaponProfile) (info.Weapon, error) {
	w := &Weapon{}
	r := float64(p.Refine)

	heal := 0.03 + r*0.01
	hpRatio := 0.03 + r*0.01
	atkPerKilo := 0.003 + r*0.001
	atkMax := 0.06 + r*0.02

	mHeal := make([]float64, attributes.EndStatType)
	mHeal[attributes.Heal] = heal
	char.AddStatMod(character.StatMod{
		Base:         modifier.NewBase("hymn-of-the-maelstrom-heal", -1),
		AffectedStat: attributes.Heal,
		Amount: func() []float64 {
			return mHeal
		},
	})

	stacks := func() int {
		n := 0
		for i := 0; i < maxStacks; i++ {
			if char.StatusIsActive(stackKey(i)) {
				n++
			}
		}
		return n
	}
	mul := func() float64 {
		m := float64(stacks())
		if char.StatusIsActive(strongKey) {
			m *= strongMul
		}
		return m
	}

	mHP := make([]float64, attributes.EndStatType)
	char.AddStatMod(character.StatMod{
		Base:         modifier.NewBase("hymn-of-the-maelstrom-hp", -1),
		AffectedStat: attributes.HPP,
		Amount: func() []float64 {
			m := mul()
			if m == 0 {
				return nil
			}
			mHP[attributes.HPP] = hpRatio * m
			return mHP
		},
	})

	// the ATK follows the active member and is recomputed from the wearer's
	// current Max HP
	w.initATK = func() {
		for _, this := range c.Player.Chars() {
			mATK := make([]float64, attributes.EndStatType)
			this.AddStatMod(character.StatMod{
				Base:         modifier.NewBase(fmt.Sprintf("hymn-of-the-maelstrom-atk-%v", char.Base.Key.String()), -1),
				AffectedStat: attributes.ATKP,
				Amount: func() []float64 {
					if c.Player.Active() != this.Index() {
						return nil
					}
					m := mul()
					if m == 0 {
						return nil
					}
					per := min(max((char.MaxHP()-minHP)*atkPerKilo/1000, 0), atkMax)
					if per == 0 {
						return nil
					}
					mATK[attributes.ATKP] = per * m
					return mATK
				},
			})
		}
	}

	c.Events.Subscribe(event.OnHeal, func(args ...any) {
		src := args[0].(*info.HealInfo)
		if src.Caller != char.Index() {
			return
		}
		free := -1
		for i := 0; i < maxStacks; i++ {
			if !char.StatusIsActive(stackKey(i)) {
				free = i
				break
			}
		}
		if free >= 0 {
			char.AddStatus(stackKey(free), stackDur, true)
			return
		}
		for i := 0; i < maxStacks; i++ {
			char.AddStatus(stackKey(i), stackDur, true)
		}
	}, fmt.Sprintf("hymn-of-the-maelstrom-heal-%v", char.Base.Key.String()))

	onReaction := func(args ...any) {
		if t, ok := args[0].(info.Target); !ok || t.Type() != info.TargettableEnemy {
			return
		}
		char.AddStatus(strongKey, strongDur, true)
	}
	c.Events.Subscribe(event.OnFrozen, onReaction, fmt.Sprintf("hymn-of-the-maelstrom-frozen-%v", char.Base.Key.String()))
	c.Events.Subscribe(event.OnStellarSwirl, onReaction, fmt.Sprintf("hymn-of-the-maelstrom-stellar-swirl-%v", char.Base.Key.String()))

	return w, nil
}

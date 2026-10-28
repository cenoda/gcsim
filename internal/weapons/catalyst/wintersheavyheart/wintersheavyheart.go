package wintersheavyheart

import (
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

// Follows Weapon_Catalyst_FrostScepter (ConfigAbility_Weapon_Catalyst2) with the
// params of EquipAffix 114437:
//   - the party's Cryo and Electro members are counted, the wearer included.
//     Cryo count = min(Cryo, 4), Electro count = min(Electro, 4 - Cryo count),
//     so Cryo members fill the cap of 4 first
//   - outside Radiance (Weapon_Catalyst_FrostScepter_Sub_Normal): EM +24 per
//     Cryo count and ATK +4.8% per Electro count
//   - in Radiance: Stellar-Conduct or Radiance: Stellar Swirl (_IS_IN_STARMODE,
//     Weapon_Catalyst_FrostScepter_Sub_Star) the buff is replaced by EM +20 and
//     Stellar-Conduct / Stellar Swirl reaction DMG +6% per counted member
//
// The game re-evaluates the counts every 2s; the party never changes here, so
// they are counted once.

const (
	maxMembers = 4

	emKey      = "winters-heavy-heart-em"
	atkKey     = "winters-heavy-heart-atk"
	stellarKey = "winters-heavy-heart-stellar"
)

type Weapon struct {
	Index  int
	core   *core.Core
	char   *character.CharWrapper
	refine int
}

func (w *Weapon) SetIndex(idx int) { w.Index = idx }

func NewWeapon(c *core.Core, char *character.CharWrapper, p info.WeaponProfile) (info.Weapon, error) {
	w := &Weapon{
		core:   c,
		char:   char,
		refine: p.Refine,
	}
	return w, nil
}

// the party is complete only once every weapon is constructed
func (w *Weapon) Init() error {
	r := float64(w.refine)
	char := w.char

	emPerCryo := 18 + r*6
	atkPerElectro := 0.036 + r*0.012
	emPerStar := 15 + r*5
	stellarPerStar := 0.045 + r*0.015

	cryo, electro := 0, 0
	for _, this := range w.core.Player.Chars() {
		switch this.Base.Element {
		case attributes.Cryo:
			cryo++
		case attributes.Electro:
			electro++
		}
	}
	cryo = min(cryo, maxMembers)
	electro = min(electro, maxMembers-cryo)
	stacks := float64(cryo + electro)
	if stacks == 0 {
		return nil
	}

	// Radiance: Stellar-Conduct inside a Polestar Field, Radiance: Stellar Swirl
	// for 8s after a Stellar Swirl
	radiance := func() bool {
		return char.StatusIsActive(reactable.PolestarFieldKey) || char.StatusIsActive(reactable.RadianceSwirlKey)
	}

	mEM := make([]float64, attributes.EndStatType)
	emNormal := emPerCryo * float64(cryo)
	emStar := emPerStar * stacks
	char.AddStatMod(character.StatMod{
		Base:         modifier.NewBase(emKey, -1),
		AffectedStat: attributes.EM,
		Amount: func() []float64 {
			if radiance() {
				mEM[attributes.EM] = emStar
			} else {
				mEM[attributes.EM] = emNormal
			}
			if mEM[attributes.EM] == 0 {
				return nil
			}
			return mEM
		},
	})

	if electro > 0 {
		mATK := make([]float64, attributes.EndStatType)
		mATK[attributes.ATKP] = atkPerElectro * float64(electro)
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBase(atkKey, -1),
			AffectedStat: attributes.ATKP,
			Amount: func() []float64 {
				if radiance() {
					return nil
				}
				return mATK
			},
		})
	}

	stellar := stellarPerStar * stacks
	char.AddReactBonusMod(character.ReactBonusMod{
		Base: modifier.NewBase(stellarKey, -1),
		Amount: func(ai info.AttackInfo) float64 {
			if !ai.AttackTag.IsStellar() {
				return 0
			}
			if !radiance() {
				return 0
			}
			return stellar
		},
	})

	return nil
}

package vodyanitsa

import (
	tmpl "github.com/genshinsim/gcsim/internal/template/character"
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	particleICD = 12 // Vodyanitsa_ElementalArt_DropBall: unique for 0.2s
	particleKey = "vodyanitsa-particle-icd"
)

type char struct {
	*tmpl.Character
	// source frame of the current Song of Ages Past; it is restarted by every skill
	songSrc int
	// Lead Vocal and Chorus stacks left (A4)
	leadVocal int
	chorus    int
	// the Stellar Vortex currently on the field and whether it became a Wandering Vortex (A1)
	vortex    info.Gadget
	wandering bool
	// expiry frames of the C4 Max HP stacks
	c4Stacks []int
}

func NewChar(s *core.Core, w *character.CharWrapper, _ info.CharacterProfile) error {
	c := char{}
	c.Character = tmpl.NewWithWrapper(s, w)

	c.EnergyMax = 60
	c.NormalHitNum = normalHitNum
	c.SkillCon = 3
	c.BurstCon = 5

	w.Character = &c

	// the engine announces Stellar Vortices only while this key is set
	// (pkg/reactable/stellarswirl_wandering.go); A1, the only source of Wandering Vortices,
	// subscribes in Init once her ascension is known
	s.Flags.Custom[reactable.WanderingVortexKey] = 1

	return nil
}

func (c *char) Init() error {
	c.a1Init()
	c.a4Init()
	c.c2Init()
	c.c4Init()
	c.c6Init()
	return nil
}

func (c *char) Condition(fields []string) (any, error) {
	switch fields[0] {
	case "song":
		return c.StatusIsActive(songKey), nil
	case "lead-vocal":
		return c.leadVocal, nil
	case "chorus":
		return c.chorus, nil
	case "wandering":
		return c.StatusIsActive(alterMarkKey), nil
	default:
		return c.Character.Condition(fields)
	}
}

// Each Horn of Spring's Call hit generates 1 Hydro particle (baseEnergy 3 in
// Vodyanitsa_ElementalArt_DropBall); the skill's first hit generates none.
func (c *char) particleCB(a info.AttackCB) {
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if c.StatusIsActive(particleKey) {
		return
	}
	c.AddStatus(particleKey, particleICD, false)
	c.Core.QueueParticle(c.Base.Key.String(), 1, attributes.Hydro, c.ParticleDelay)
}

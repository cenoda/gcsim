package lohen

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Two hits (A and B), hit lag on B only, from the game's ability data.
// Frames are estimated: CA -> N1 (27) plus N1 -> CA (16) gives a 43-frame N1C loop, so that
// a 13s Masterstroke fits 15-16 N1C after the skill and 17 only with near frame-perfect
// input, as the KQM Lohen quick guide reports. Other cancels follow Flins.
var (
	chargeFrames          []int
	chargeHitmarks        = []int{12, 20}
	chargeHitlagHaltFrame = []float64{0, 0.02}
)

func init() {
	chargeFrames = frames.InitAbilSlice(50)
	chargeFrames[action.ActionAttack] = 27
	chargeFrames[action.ActionSkill] = 27
	chargeFrames[action.ActionBurst] = 27
	chargeFrames[action.ActionDash] = chargeHitmarks[1]
	chargeFrames[action.ActionJump] = chargeHitmarks[1]
	chargeFrames[action.ActionWalk] = 45
	chargeFrames[action.ActionSwap] = 27
}

func (c *char) ChargeAttack(p map[string]int) (action.Info, error) {
	masterstroke := c.masterstrokeActive()
	for i, hitmark := range chargeHitmarks {
		ai := info.AttackInfo{
			ActorIndex:         c.Index(),
			Abil:               fmt.Sprintf("Charge %v", i),
			AttackTag:          attacks.AttackTagExtra,
			ICDTag:             attacks.ICDTagNormalAttack,
			ICDGroup:           attacks.ICDGroupLohenNormalAttack,
			StrikeType:         attacks.StrikeTypeSlash,
			Element:            attributes.Physical,
			Durability:         25,
			Mult:               charge[c.TalentLvlAttack()],
			HitlagFactor:       0.01,
			HitlagHaltFrames:   chargeHitlagHaltFrame[i] * 60,
			CanBeDefenseHalted: false,
		}
		if masterstroke {
			ai.Abil = fmt.Sprintf("Masterstroke Charge %v", i)
			ai.Element = attributes.Cryo
			ai.Mult = skillCharge[c.skillLvl()]
		}
		ap := combat.NewBoxHitOnTarget(c.Core.Combat.Player(), nil, 3.5, 4.8)
		c.QueueCharTask(func() {
			c.Core.QueueAttack(ai, ap, 0, 0, c.joyCB(joySrcChargeA+i), c.particleCB, c.c2CB)
		}, hitmark)
	}

	return action.Info{
		Frames:          frames.NewAbilFunc(chargeFrames),
		AnimationLength: chargeFrames[action.InvalidAction],
		CanQueueAfter:   chargeHitmarks[1],
		State:           action.ChargeAttackState,
	}, nil
}

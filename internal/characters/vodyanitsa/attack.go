package vodyanitsa

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Element, gauge, ICD and hit shape are from the game's data (Bullet_Vodyanitsa_Attack01-03 are
// 20 m/s bullets with a 0.7 m hit, Avatar_Vodyanitsa_Attack04 an AoE). No frame count exists:
// hitmarks and cancels are Sucrose's, a counted 4-hit catalyst with the same body type, and the
// bullets add the usual catalyst travel.
var (
	attackFrames   [][]int
	attackHitmarks = []int{17, 18, 28, 28}
	attackRadius   = []float64{0.7, 0.7, 0.7, 2.5}
)

const normalHitNum = 4

func init() {
	attackFrames = make([][]int, normalHitNum)

	attackFrames[0] = frames.InitNormalCancelSlice(attackHitmarks[0], 28)
	attackFrames[0][action.ActionAttack] = 17
	attackFrames[0][action.ActionCharge] = 20

	attackFrames[1] = frames.InitNormalCancelSlice(attackHitmarks[1], 38)
	attackFrames[1][action.ActionAttack] = 26
	attackFrames[1][action.ActionCharge] = 18

	attackFrames[2] = frames.InitNormalCancelSlice(attackHitmarks[2], 46)
	attackFrames[2][action.ActionAttack] = 33
	attackFrames[2][action.ActionCharge] = 28

	attackFrames[3] = frames.InitNormalCancelSlice(attackHitmarks[3], 65)
	attackFrames[3][action.ActionAttack] = 51
	attackFrames[3][action.ActionCharge] = 54
}

func (c *char) Attack(p map[string]int) (action.Info, error) {
	travel, ok := p["travel"]
	if !ok {
		travel = 10
	}

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       fmt.Sprintf("Normal %v", c.NormalCounter),
		AttackTag:  attacks.AttackTagNormal,
		ICDTag:     attacks.ICDTagNormalAttack,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Hydro,
		Durability: 25,
		Mult:       attack[c.NormalCounter][c.TalentLvlAttack()],
	}

	hitmark := attackHitmarks[c.NormalCounter]
	dmgDelay := hitmark + travel
	if c.NormalCounter == 3 {
		// N4 is an AoE at the target, not a bullet
		dmgDelay = hitmark
	}
	ap := combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, attackRadius[c.NormalCounter])
	c.Core.QueueAttack(ai, ap, hitmark, dmgDelay)

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, attackFrames),
		AnimationLength: attackFrames[c.NormalCounter][action.InvalidAction],
		CanQueueAfter:   hitmark,
		State:           action.NormalAttackState,
	}, nil
}

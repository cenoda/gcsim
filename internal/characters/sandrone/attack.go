package sandrone

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Hit lag (0.1, 0.1, 0.12s), strike type (Blunt), element (none) and hit count are from the game's
// ConfigAvatar. Hitmarks and cancels are estimated from Dehya's N1-N3; her normal attacks are not
// used in the community rotations.
var (
	attackFrames   [][]int
	attackHitmarks = []int{22, 26, 30}
	attackHitlag   = []float64{0.1, 0.1, 0.12}
	attackRadius   = []float64{2.5, 2.5, 3}
)

const normalHitNum = 3

func init() {
	attackFrames = make([][]int, normalHitNum)
	attackFrames[0] = frames.InitNormalCancelSlice(attackHitmarks[0], 31)
	attackFrames[1] = frames.InitNormalCancelSlice(attackHitmarks[1], 34)
	attackFrames[2] = frames.InitNormalCancelSlice(attackHitmarks[2], 70)
	attackFrames[2][action.ActionAttack] = 62
}

func (c *char) Attack(p map[string]int) (action.Info, error) {
	ai := info.AttackInfo{
		ActorIndex:         c.Index(),
		Abil:               fmt.Sprintf("Normal %v", c.NormalCounter),
		AttackTag:          attacks.AttackTagNormal,
		ICDTag:             attacks.ICDTagNormalAttack,
		ICDGroup:           attacks.ICDGroupDefault,
		StrikeType:         attacks.StrikeTypeBlunt,
		Element:            attributes.Physical,
		Durability:         25,
		Mult:               attack[c.NormalCounter][c.TalentLvlAttack()],
		HitlagFactor:       0.01,
		HitlagHaltFrames:   attackHitlag[c.NormalCounter] * 60,
		CanBeDefenseHalted: true,
	}
	ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 1}, attackRadius[c.NormalCounter])
	c.Core.QueueAttack(ai, ap, attackHitmarks[c.NormalCounter], attackHitmarks[c.NormalCounter])

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, attackFrames),
		AnimationLength: attackFrames[c.NormalCounter][action.InvalidAction],
		CanQueueAfter:   attackHitmarks[c.NormalCounter],
		State:           action.NormalAttackState,
	}, nil
}

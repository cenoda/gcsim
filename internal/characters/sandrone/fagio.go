package sandrone

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Fagio is not a gadget in the game: its sweeping fire, condensed beams and Power Overdrive shots
// are bullets fired by Sandrone during her charged attack, driven by the Decoding Power global
// value (Avatar_MarionetteNew_SpecialExAtkDriver, Avatar_Marionette_FrostValue_Driver). All
// timings below are from that ability data and the bullets' gadget configs.
const (
	powerMax          = 100
	overdriveEnter    = 99.5 // Marionette_FrostValue_Overload_Applier step
	overdriveExit     = 50
	decodingTick      = 12 // 0.2s: +3.2 Decoding Power (MarionetteTickAddFrostValue)
	decodingGain      = 3.2
	beamTick          = 60 // 1s: +17 Decoding Power and the beam counter (Avatar_Marionette_TriggerBullet)
	beamGain          = 17
	forceDelay        = 15 // 0.25s: UNIQUE_MarionetteForceSet100 sets 100 after the 3rd (C1: 6th) count
	beamFirst         = 51 // 0.85s, then every 1s (MarionetteCylinderAttackGrandpa/Father)
	beamInterval      = 60
	sweepInterval     = 18 // 0.3s hit interval of the sweeping fire bullet
	overdriveInterval = 24 // 0.4s (Avatar_Marionette_TriggerBulletWeak)
	decayInterval     = 30 // 0.5s: -2.5 on field, -7.5 off field
	decayAmount       = 2.5
	decayOffField     = 3
	tacticsDur        = 60 * 60
	tacticsMax        = 10
)

// C6 Condensed Cluster Beam: one bullet, alive 3.1s, first hit after 0.5s and then every 0.8s
var c6Hitmarks = []int{30, 78, 126, 174}

func (c *char) fagioInit() {
	c.Core.Tasks.Add(c.decayTick, decayInterval)
}

func (c *char) gainRatio() float64 {
	if c.Base.Cons >= 1 {
		return 0.5
	}
	return 1
}

// beam count at which Decoding Power is set to 100; at C6 the counter stops at 2 and the
// third count starts the Condensed Cluster Beam instead
func (c *char) forcedCount() int {
	switch {
	case c.Base.Cons >= 6:
		return -1
	case c.Base.Cons >= 1:
		return 6
	default:
		return 3
	}
}

func (c *char) addPower(v float64) {
	c.power = min(max(c.power+v, 0), powerMax)
	switch {
	case !c.overdrive && c.power >= overdriveEnter:
		c.overdrive = true
		c.Core.Log.NewEvent("fagio enters power overdrive", glog.LogCharacterEvent, c.Index())
		if c.caActive(c.caSrc) {
			c.startOverdriveShots(c.caSrc)
		}
	case c.overdrive && c.power <= overdriveExit:
		c.overdrive = false
		c.Core.Log.NewEvent("fagio exits power overdrive", glog.LogCharacterEvent, c.Index())
	}
}

// reducePower lowers Decoding Power and counts the loss towards Refined Tactics
func (c *char) reducePower(v float64) {
	lost := min(v, c.power)
	c.addPower(-v)
	if lost > 0 {
		c.addLost(lost)
	}
}

// Refined Tactics: 1 stack per 10 Decoding Power lost, for 60s from the last loss, max 10
func (c *char) addLost(v float64) {
	c.powerLost = min(c.powerLost+v, tacticsMax*10)
	src := c.Core.F
	c.powerLostSrc = src
	c.Core.Tasks.Add(func() {
		if c.powerLostSrc == src {
			c.powerLost = 0
		}
	}, tacticsDur)
}

func (c *char) refinedTactics() int {
	return min(int(c.powerLost/10), tacticsMax)
}

// Decoding Power decreases while Sandrone is neither charging nor hovering, 3 times as fast
// while she is off field
func (c *char) decayTick() {
	c.Core.Tasks.Add(c.decayTick, decayInterval)
	if c.caSrc != 0 || c.hoverSrc != 0 || c.power <= 0 {
		return
	}
	amt := decayAmount
	if c.Core.Player.Active() != c.Index() {
		amt *= decayOffField
	}
	c.reducePower(amt)
}

func (c *char) caActive(src int) bool {
	return src != 0 && c.caSrc == src
}

func (c *char) decodingActive(src int) bool {
	return c.caActive(src) && !c.overdrive
}

// overdriveAfter returns how many frames after the start of Decoding mode Fagio enters Power
// Overdrive, starting from the current Decoding Power, following the same rules as decoding
func (c *char) overdriveAfter() int {
	if c.overdrive {
		return 0
	}
	p := c.power
	count := 0
	forced := -1
	for f := 1; f <= maxHold; f++ {
		if f%decodingTick == 0 {
			p += decodingGain * c.gainRatio()
		}
		if f%beamTick == 0 {
			if c.Base.Cons < 6 || count <= 1 {
				count++
				if count == c.forcedCount() {
					forced = f + forceDelay
				}
			}
			p += beamGain * c.gainRatio()
		}
		if f == forced {
			p += powerMax
		}
		if p >= overdriveEnter {
			return f
		}
	}
	return maxHold
}

func (c *char) startDecoding(src int) {
	if c.overdrive {
		c.startOverdriveShots(src)
		return
	}
	c.c2Layers = 0
	count := 0
	var tick func()
	tick = func() {
		if !c.decodingActive(src) {
			return
		}
		c.addPower(decodingGain * c.gainRatio())
		c.Core.Tasks.Add(tick, decodingTick)
	}
	c.Core.Tasks.Add(tick, decodingTick)

	var counter func()
	counter = func() {
		if !c.decodingActive(src) {
			return
		}
		if c.Base.Cons < 6 || count <= 1 {
			count++
			if count == c.forcedCount() {
				gen := c.forceGen
				c.Core.Tasks.Add(func() {
					// the skill removes the pending set (MarionetteNew_ElementalArt_ReduceFrost)
					if c.forceGen != gen {
						return
					}
					c.addPower(powerMax)
				}, forceDelay)
			}
		} else if count == 2 {
			count++
			c.c6ClusterBeam(src)
		}
		c.addPower(beamGain * c.gainRatio())
		c.Core.Tasks.Add(counter, beamTick)
	}
	c.Core.Tasks.Add(counter, beamTick)

	var fire func()
	fire = func() {
		if !c.decodingActive(src) {
			return
		}
		c.c2AddLayer()
		c.fireBeam("Condensed Beam", beam, beamSSC, beamSSw, c.TalentLvlAttack(), true)
		c.Core.Tasks.Add(fire, beamInterval)
	}
	c.Core.Tasks.Add(fire, beamFirst)
}

// fireBeam deals a beam hit: Cryo damage, or direct Stellar-Conduct / Stellar Swirl damage in
// the matching Radiance state, which replaces it (Avatar_Marionette_ExtraAttackBullet_01/_Loop)
func (c *char) fireBeam(abil string, normal, ssc, ssw []float64, lvl int, particle bool) {
	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       abil,
		AttackTag:  attacks.AttackTagExtra,
		ICDTag:     attacks.ICDTagExtraAttack,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeBlunt,
		Element:    attributes.Cryo,
		Durability: 25,
	}
	if normal != nil {
		ai.Mult = normal[lvl]
	} else {
		ai.Mult = c6ClusterMult
	}
	radiance := c.getRadiance()
	if radiance != radianceNone {
		c.stellarInfo(&ai, abil, radiance, ssc, ssw, lvl)
	}
	snap := c.Snapshot(&ai)
	if radiance != radianceNone {
		snap.Stats[attributes.CD] += c.c2CritDMG()
	}
	ap := combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 2.5)
	// the Stellar Glimmer versions always count for particles, the normal C6 cluster beam does not
	if particle || radiance != radianceNone {
		c.Core.QueueAttackWithSnap(ai, snap, ap, 0, c.particleCB, c.c4CB)
		return
	}
	c.Core.QueueAttackWithSnap(ai, snap, ap, 0, c.c4CB)
}

// sweeping fire: from 0.65 of the charged attack's entry, while in Decoding mode
func (c *char) startSweep(src int) {
	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Sweeping Fire",
		AttackTag:  attacks.AttackTagExtra,
		ICDTag:     attacks.ICDTagSandroneMachineGun,
		ICDGroup:   attacks.ICDGroupSandroneMachineGun,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       sweep[c.TalentLvlAttack()],
	}
	var hit func()
	hit = func() {
		if !c.decodingActive(src) {
			return
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 1.8)
		c.Core.QueueAttack(ai, ap, 0, 0)
		c.Core.Tasks.Add(hit, sweepInterval)
	}
	c.Core.Tasks.Add(hit, sweepInterval)
}

// Power Overdrive: a shot on entering it and every 0.4s while the charged attack is held
func (c *char) startOverdriveShots(src int) {
	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Power Overdrive Shot",
		AttackTag:  attacks.AttackTagExtra,
		ICDTag:     attacks.ICDTagSandroneMachineGun,
		ICDGroup:   attacks.ICDGroupSandroneMachineGun,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       overdrive[c.TalentLvlAttack()],
	}
	var shot func()
	shot = func() {
		if !c.caActive(src) || !c.overdrive {
			return
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 0.5)
		c.Core.QueueAttack(ai, ap, 0, 0)
		c.Core.Tasks.Add(shot, overdriveInterval)
	}
	shot()
}

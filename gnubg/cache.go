package gnubg

import "fmt"

type _CacheNodeDetail struct {
	key          _PositionKey
	nEvalContext int
	ar           [6]float32
}

type _CacheNode struct {
	nd_primary   _CacheNodeDetail
	nd_secondary _CacheNodeDetail
}

type _HashKey uint32

type _EvalCache struct {
	entries  []_CacheNode
	size     int
	hashMask _HashKey
}

func cacheCreate(pc *_EvalCache, s int) error {
	// #if CACHE_STATS
	//     pc->cLookup = 0;
	//     pc->cHit = 0;
	//     pc->nAdds = 0;
	// #endif

	if s > 1<<31 {
		return fmt.Errorf("too large")
	}

	pc.size = s
	/* adjust size to smallest power of 2 GE to s */
	for (s & (s - 1)) != 0 {
		s &= (s - 1)
	}
	if s < pc.size {
		pc.size = 2 * s
	} else {
		pc.size = s
	}
	pc.hashMask = _HashKey((pc.size >> 1) - 1)

	pc.entries = make([]_CacheNode, pc.size/2)

	cacheFlush(pc)
	return nil
}

func cacheDestroy(pc *_EvalCache) {
	pc.entries = nil
}

func cacheFlush(pc *_EvalCache) {
	for k := 0; k < pc.size/2; k++ {
		pc.entries[k].nd_primary.key.data[0] = -1
		pc.entries[k].nd_secondary.key.data[0] = -1
	}
}

func cacheLookup(pc *_EvalCache, e *_CacheNodeDetail, arOut *[_NUM_OUTPUTS]float32, arCubeful *float32) (hit bool, l _HashKey) {
	l = getHashKey(pc.hashMask, e)

	// #if CACHE_STATS
	//     ++pc->cLookup;
	// #endif

	if !pc.entries[l].nd_primary.key.equals(e.key) || pc.entries[l].nd_primary.nEvalContext != e.nEvalContext { /* Not in primary slot */
		if !pc.entries[l].nd_secondary.key.equals(e.key) || pc.entries[l].nd_secondary.nEvalContext != e.nEvalContext { /* Cache miss */
			return
		} else { /* Found in second slot, promote "hot" entry */
			tmp := pc.entries[l].nd_primary

			pc.entries[l].nd_primary = pc.entries[l].nd_secondary
			pc.entries[l].nd_secondary = tmp
		}
	}

	/* Cache hit */
	hit = true
	copy((*arOut)[:], pc.entries[l].nd_primary.ar[:_NUM_OUTPUTS])
	if arCubeful != nil {
		*arCubeful = pc.entries[l].nd_primary.ar[5] /* Cubeful equity stored in slot 5 */
	}
	// #if CACHE_STATS
	//     ++pc->cHit;
	// #endif

	return
}

func getHashKey(hashMask _HashKey, e *_CacheNodeDetail) _HashKey {
	hash := _HashKey(e.nEvalContext)

	hash *= 0xcc9e2d51
	hash = (hash << 15) | (hash >> (32 - 15))
	hash *= 0x1b873593

	hash = (hash << 13) | (hash >> (32 - 13))
	hash = hash*5 + 0xe6546b64

	for i := 0; i < 7; i++ {
		k := _HashKey(e.key.data[i])

		k *= 0xcc9e2d51
		k = (k << 15) | (k >> (32 - 15))
		k *= 0x1b873593

		hash ^= k
		hash = (hash << 13) | (hash >> (32 - 13))
		hash = hash*5 + 0xe6546b64
	}

	/* Real MurmurHash3 has a "hash ^= len" here,
	 * but for us len is constant. Skip it */

	hash ^= hash >> 16
	hash *= 0x85ebca6b
	hash ^= hash >> 13
	hash *= 0xc2b2ae35
	hash ^= hash >> 16

	return hash & hashMask
}

func cacheAdd(pc *_EvalCache, e *_CacheNodeDetail, l _HashKey) {
	pc.entries[l].nd_secondary = pc.entries[l].nd_primary
	pc.entries[l].nd_primary = *e

	// #if CACHE_STATS
	//     ++pc->nAdds;
	// #endif
}

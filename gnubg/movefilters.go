package gnubg

var _MOVEFILTER_TINY = [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter{
	{_MoveFilter{0, 5, 0.08}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 5, 0.08}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 5, 0.08}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.02}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 5, 0.08}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.02}, _MoveFilter{-1, 0, 0}},
}

var _MOVEFILTER_NARROW = [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter{
	{_MoveFilter{0, 8, 0.12}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.12}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.12}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.03}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.12}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.03}, _MoveFilter{-1, 0, 0}},
}

var _MOVEFILTER_NORMAL = [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter{
	{_MoveFilter{0, 8, 0.16}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.16}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.16}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.04}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 8, 0.16}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 2, 0.04}, _MoveFilter{-1, 0, 0}},
}

var _MOVEFILTER_LARGE = [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter{
	{_MoveFilter{0, 16, 0.32}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 16, 0.32}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 16, 0.32}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 4, 0.08}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 16, 0.32}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 4, 0.08}, _MoveFilter{-1, 0, 0}},
}

var _MOVEFILTER_HUGE = [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter{
	{_MoveFilter{0, 20, 0.44}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 20, 0.44}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 0, 0}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 20, 0.44}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 6, 0.11}, _MoveFilter{0, 0, 0}},
	{_MoveFilter{0, 20, 0.44}, _MoveFilter{-1, 0, 0}, _MoveFilter{0, 6, 0.11}, _MoveFilter{-1, 0, 0}},
}

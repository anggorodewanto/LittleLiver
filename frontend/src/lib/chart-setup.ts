import {
	Chart,
	LineController,
	BarController,
	ScatterController,
	CategoryScale,
	LinearScale,
	TimeScale,
	PointElement,
	LineElement,
	BarElement,
	Filler,
	Legend,
	Tooltip
} from 'chart.js';

Chart.register(
	LineController,
	BarController,
	ScatterController,
	CategoryScale,
	LinearScale,
	TimeScale,
	PointElement,
	LineElement,
	BarElement,
	Filler,
	Legend,
	Tooltip
);

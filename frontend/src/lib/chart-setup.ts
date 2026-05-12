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
	Tooltip,
	SubTitle
} from 'chart.js';
import zoomPlugin from 'chartjs-plugin-zoom';

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
	Tooltip,
	SubTitle,
	zoomPlugin
);

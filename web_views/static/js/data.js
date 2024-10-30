function initializeChart(jsData) {
    if (!window.Chart) {
        console.warn("Chart.js not loaded.");
        return;
    }

    const rmseLabels = jsData.RMSEEvaluationHeaders.slice(1)
    const rmseValues = jsData.RMSEEvaluationValues

    const rmseData = {
        labels: rmseValues.map(value => value[0]),
        datasets: rmseLabels.map((label, index) => {
            return {
                label: label,
                data: rmseValues.map(value => parseFloat(value[index + 1])),
                borderColor: `hsl(${index * 360 / rmseLabels.length}, 50%, 40%)`,
                backgroundColor: `rgba(0, 0, 0, 0)`,
                fill: false,
                tension: 0.1
            };
        })
    };

    // Chart configuration
    const rmseConfig = {
        type: 'line',
        data: rmseData,
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'top',
                },
                title: {
                    display: true,
                    text: 'RMSE for Weather Parameter VAR Prediction'
                }
            },
            scales: {
                x: {
                    title: {
                        display: true,
                        text: 'Train-Test Ratio'
                    }
                },
                y: {
                    title: {
                        display: true,
                        text: 'RMSE Values'
                    }
                }
            }
        }
    };

    // Create the chart
    new Chart(
        document.getElementById('rmseTable'),
        rmseConfig
    );
}
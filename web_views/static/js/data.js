function initializeChart(jsData) {
    if (!window.Chart) {
        console.warn("Chart.js not loaded.");
        return;
    }

    const nasaLabels = jsData.NasaHeaders.slice(1)
    const nasaValues = jsData.NasaValues
    const nrmseLabels = jsData.NRMSEEvaluationHeaders.slice(1)
    const nrmseValues = jsData.NRMSEEvaluationValues

    // Create the chart
    new Chart(
        document.getElementById('nasaTable'),
        getNasaConfig(nasaLabels, nasaValues)
    )
    new Chart(
        document.getElementById('nrmseTable'),
        getNRMSEConfig(nrmseLabels, nrmseValues)
    )
}

function getNasaConfig(labels, values) {
    const data = {
        labels: values.map(value => value[0]),
        datasets: labels.map((label, index) => {
            return {
                label: label,
                data: values.map(value => parseFloat(value[index + 1])),
                borderColor: `hsl(${index * 360 / labels.length}, 50%, 40%)`,
                backgroundColor: `rgba(0, 0, 0, 0)`,
                fill: false,
                tension: 0.1,
                pointRadius: 0,
                pointHoverRadius: 0
            };
        })
    };

    // Chart configuration
    const config = {
        type: 'line',
        data: data,
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'top',
                },
                title: {
                    display: true,
                    text: 'NASA POWER API Weather Parameter Data'
                }
            },
            scales: {
                x: {
                    title: {
                        display: true,
                        text: 'Date'
                    },
                    ticks: {
                        maxTicksLimit: 8
                    }
                },
                y: {
                    title: {
                        display: true,
                        text: 'Parameter Values'
                    }
                }
            }
        }
    }

    return config
}

function getNRMSEConfig(labels, values) {
    const data = {
        labels: values.map(value => value[0]),
        datasets: labels.map((label, index) => {
            return {
                label: label,
                data: values.map(value => parseFloat(value[index + 1])),
                borderColor: `hsl(${index * 360 / labels.length}, 50%, 40%)`,
                backgroundColor: `rgba(0, 0, 0, 0)`,
                fill: false,
                tension: 0.1
            };
        })
    };

    // Chart configuration
    const config = {
        type: 'line',
        data: data,
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'top',
                },
                title: {
                    display: true,
                    text: 'NRMSE for Weather Parameter VAR Prediction'
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
                        text: 'NRMSE Values'
                    }
                }
            }
        }
    }

    return config
}

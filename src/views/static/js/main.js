async function deleteChart(name, version) {
    if (!confirm('Are you sure you want to delete this version?')) {
        return;
    }

    try {
        const response = await fetch(`/chart/${name}/${version}`, {
            method: 'DELETE',
        });
        if (response.ok) {
            // Trouver la carte à mettre à jour
            const chartCard = document.querySelector(`[data-chart-name="${name}"]`);
            if (chartCard) {
                // Récupérer les versions mises à jour
                const updatedVersions = await fetchChartVersions(name);
                if (updatedVersions.length === 0) {
                    // Si plus de versions, supprimer la carte
                    chartCard.remove();
                } else {
                    // Sinon mettre à jour la liste des versions
                    updateVersionsList(chartCard, updatedVersions);
                }
            }
        } else {
            const error = await response.text();
            alert(`Failed to delete chart: ${error}`);
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error deleting chart');
    }
}

// Récupère les versions d'un chart
async function fetchChartVersions(name) {
    try {
        const response = await fetch(`/chart/${name}/versions`);
        if (response.ok) {
            return await response.json();
        }
        return [];
    } catch (error) {
        console.error('Error fetching versions:', error);
        return [];
    }
}
function switchVersion(chartName, version) {
    const card = document.querySelector(`[data-chart-name="${chartName}"]`);
    if (!card) return;

    // Mise à jour des URLs des actions
    const infoLink = card.querySelector('.icon-info').parentElement;
    const downloadLink = card.querySelector('.icon-download').parentElement;
    const deleteLink = card.querySelector('.icon-delete').parentElement;

    infoLink.href = `/chart/${chartName}/${version}/details`;
    downloadLink.href = `/chart/${chartName}/${version}`;
    deleteLink.onclick = () => deleteChart(chartName, version);

    // Mise à jour des détails de la version
    const versions = chartVersions[chartName] || [];
    const currentVersion = versions.find(v => v.version === version);
    if (currentVersion) {
        card.querySelector('.version-app').textContent = currentVersion.appVersion || 'N/A';
        card.querySelector('.version-description').textContent = currentVersion.description || '';
    }
}

// Au chargement de la page, initialiser les versions
let chartVersions = {};
document.addEventListener('DOMContentLoaded', async () => {
    const cards = document.querySelectorAll('[data-chart-name]');
    for (const card of cards) {
        const chartName = card.dataset.chartName;
        const select = card.querySelector('select');
        if (select && select.value) {
            const version = select.value;
            switchVersion(chartName, version);
        }
    }
});

// Met à jour la liste des versions dans la carte
function updateVersionsList(cardElement, versions) {
    const versionsList = cardElement.querySelector('.versions-list');
    if (versionsList) {
        versionsList.innerHTML = versions.map(version => `
            <div class="version-item border-t pt-4">
                <div class="space-y-2 text-sm text-gray-600">
                    <p><span class="font-semibold">Version:</span> ${version.version}</p>
                    ${version.appVersion ? 
                        `<p><span class="font-semibold">App Version:</span> ${version.appVersion}</p>` 
                        : ''}
                </div>
                <p class="mt-2 text-gray-700">${version.description}</p>
                
                <div class="mt-4 flex justify-end gap-2">
                    <a href="/chart/${version.name}/${version.version}/details" 
                       class="tooltip-trigger"
                       data-tooltip="View chart details">
                        <i class="material-icons icon-info">info</i>
                    </a>
                    <a href="/chart/${version.name}/${version.version}" 
                       class="tooltip-trigger"
                       data-tooltip="Download chart package">
                        <i class="material-icons icon-download">download</i>
                    </a>
                    <a href="#" 
                       onclick="deleteChart('${version.name}', '${version.version}')" 
                       class="tooltip-trigger"
                       data-tooltip="Delete this version">
                        <i class="material-icons icon-delete">delete</i>
                    </a>
                </div>
            </div>
        `).join('');
    }
}
/**
 * Helm Portal - Main JavaScript File
 * Ce fichier contient toutes les fonctionnalités JavaScript pour le portail Helm Charts
 */

// ⚙️ Gestion des modales
/**
 * Affiche la modale avec un message personnalisé
 * @param {string} message - Le message à afficher dans la modale
 * @param {boolean} isError - Indique s'il s'agit d'une erreur (rouge) ou d'un succès (vert)
 */
function showModal(message, isError = true) {
    // ⚠️ Debug - Vérifier si la fonction est appelée
    console.log('showModal called:', message, isError);
    
    const modal = document.getElementById('errorModal');
    const content = document.getElementById('errorModalContent');
    const title = modal.querySelector('h3');
    
    // Mettre à jour le contenu et l'apparence
    content.textContent = message;
    
    if (isError) {
        title.textContent = 'Erreur';
        title.classList.remove('text-green-600');
        title.classList.add('text-red-600');
    } else {
        title.textContent = 'Succès';
        title.classList.remove('text-red-600');
        title.classList.add('text-green-600');
    }
    
    // Afficher la modale - s'assurer qu'elle est visible
    modal.classList.remove('hidden');
    modal.style.display = 'flex';
    
    // ⚠️ Debug - Vérifier l'état de la modale après tentative d'affichage
    console.log('Modal state after show:', modal.classList, modal.style.display);
}

/**
 * Ferme la modale
 */
function closeErrorModal() {
    const modal = document.getElementById('errorModal');
    modal.classList.add('hidden');
    modal.style.display = 'none';
}

// 🔄 Gestion des API et requêtes
/**
 * Gestionnaire d'erreur générique pour les appels fetch
 * @param {Response} response - La réponse de l'API
 * @returns {Promise} - Retourne les données JSON ou lève une erreur
 */
function handleFetchError(response) {
    if (!response.ok) {
        return response.json().then(data => {
            throw new Error(data.error || 'Une erreur s\'est produite');
        });
    }
    return response.json();
}

/**
 * Récupère les versions d'un chart spécifique
 * @param {string} name - Le nom du chart
 * @returns {Promise<Array>} - Les versions du chart ou un tableau vide en cas d'erreur
 */
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

// 💾 Fonctionnalités de sauvegarde
/**
 * Effectue une sauvegarde du système
 * @returns {Promise<void>}
 */
async function performBackup() {
    try {
        const response = await fetch('/backup', {
            method: 'POST'
        });
        
        const data = await handleFetchError(response);
        showModal('Backup réalisé avec succès: ' + data.message, false);
    } catch (error) {
        console.error('Erreur:', error);
        showModal('Erreur lors du backup: ' + error.message);
    }
}

/**
 * Vérifie si la fonctionnalité de backup est activée
 * @returns {Promise<void>}
 */
async function checkBackupStatus() {
    try {
        const response = await fetch('/backup/status');
        const data = await response.json();
        
        const backupForm = document.getElementById('backupButton').closest('form');
        if (!data.enabled) {
            backupForm.style.display = 'none';
        }
    } catch (error) {
        console.error('Error fetching backup status:', error);
    }
}

// 📊 Gestion des charts
/**
 * Bascule vers une autre version d'un chart
 * @param {string} chartName - Le nom du chart
 * @param {string} version - La version à afficher
 */
function switchVersion(chartName, version) {
    const card = document.querySelector(`[data-chart-name="${chartName}"]`);
    if (!card) return;

    // Mise à jour des URLs des actions
    const infoLink = card.querySelector('.icon-info').parentElement;
    const downloadLink = card.querySelector('.icon-download').parentElement;
    const deleteLink = card.querySelector('.icon-delete').parentElement;

    infoLink.href = `/chart/${chartName}/${version}/details`;
    downloadLink.href = `/chart/${chartName}/${version}`;
    
    // Réinitialiser le gestionnaire d'événements pour le bouton de suppression
    deleteLink.onclick = function() { deleteChart(chartName, version); };

    // Si nous avons des données de version en cache, mettre à jour les détails
    if (window.chartVersions && window.chartVersions[chartName]) {
        const currentVersion = window.chartVersions[chartName].find(v => v.version === version);
        if (currentVersion) {
            const appVersionElem = card.querySelector('.version-details p span');
            const descriptionElem = card.querySelector('.description');
            
            if (appVersionElem && appVersionElem.nextSibling) {
                appVersionElem.nextSibling.textContent = ' ' + (currentVersion.appVersion || 'N/A');
            }
            
            if (descriptionElem) {
                descriptionElem.textContent = currentVersion.description || '';
            }
        }
    }
}

/**
 * Supprime une version spécifique d'un chart
 * @param {string} name - Le nom du chart
 * @param {string} version - La version à supprimer
 * @returns {Promise<void>}
 */
async function deleteChart(name, version) {
    if (!confirm('Are you sure you want to delete this version?')) {
        return;
    }

    try {
        const response = await fetch(`/chart/${name}/${version}`, {
            method: 'DELETE',
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || 'Failed to delete chart');
        }
        
        // Trouver la carte à mettre à jour
        const chartCard = document.querySelector(`[data-chart-name="${name}"]`);
        if (chartCard) {
            // Récupérer les versions mises à jour
            const updatedVersions = await fetchChartVersions(name);
            if (updatedVersions.length === 0) {
                // Si plus de versions, supprimer la carte
                chartCard.remove();
                showModal(`Chart ${name} a été complètement supprimé`, false);
            } else {
                // Sinon, mettre à jour l'interface si nécessaire
                updateChart(chartCard, name, updatedVersions);
                showModal(`Version ${version} du chart ${name} supprimée avec succès`, false);
            }
        }
    } catch (error) {
        console.error('Error:', error);
        showModal(`Erreur lors de la suppression: ${error.message}`);
    }
}

/**
 * Met à jour l'affichage d'une carte chart après modification des versions
 * @param {HTMLElement} cardElement - L'élément DOM de la carte
 * @param {string} chartName - Le nom du chart
 * @param {Array} versions - Les versions disponibles
 */
function updateChart(cardElement, chartName, versions) {
    // Mise à jour du sélecteur de version si présent
    const select = cardElement.querySelector('select');
    if (select) {
        // Sauvegarder l'ancienne valeur sélectionnée si possible
        const oldValue = select.value;
        
        // Créer les nouvelles options
        select.innerHTML = versions.map(v => 
            `<option value="${v.version}">Version: ${v.version}</option>`
        ).join('');
        
        // Sélectionner la première version disponible
        const newVersion = versions[0].version;
        select.value = newVersion;
        
        // Mettre à jour les détails affichés
        switchVersion(chartName, newVersion);
    }
    
    // Stocker les versions dans le cache
    if (!window.chartVersions) window.chartVersions = {};
    window.chartVersions[chartName] = versions;
}

// 🚀 Initialisation
document.addEventListener('DOMContentLoaded', function () {
    console.log('DOM loaded'); // Debug
    
    // Vérifier le statut de la fonctionnalité de backup
    checkBackupStatus();
    
    // Initialiser le gestionnaire d'événements pour le formulaire d'upload
    const uploadForm = document.getElementById('uploadForm');
    if (uploadForm) {
        uploadForm.addEventListener('submit', function () {
            const fileInput = this.querySelector('input[type="file"]');
            if (fileInput.files.length > 0) {
                fileInput.insertAdjacentHTML('afterend',
                    '<span class="ml-2 text-blue-600">⏳ Uploading ' + fileInput.files[0].name + '...</span>');
            }
        });
    }
    
    // Sélectionner les boutons de fermeture de la modale par leur position plutôt que par l'attribut onclick
    const modalCloseIcon = document.querySelector('#errorModal .material-icons');
    const modalCloseButton = document.querySelector('#errorModal .bg-blue-600');
    
    if (modalCloseIcon) {
        modalCloseIcon.addEventListener('click', function() {
            closeErrorModal();
        });
    }
    
    if (modalCloseButton) {
        modalCloseButton.addEventListener('click', function() {
            closeErrorModal();
        });
    }
    
    // Remplacer le gestionnaire d'événement du bouton de backup
    const backupButton = document.getElementById('backupButton');
    if (backupButton) {
        // Supprimer l'attribut onclick pour éviter les conflits
        backupButton.removeAttribute('onclick');
        backupButton.addEventListener('click', function(e) {
            e.preventDefault();
            performBackup();
            return false;
        });
    }
    
 
    // Initialiser le cache des versions
    window.chartVersions = {};
    
    // Pré-charger les versions pour chaque chart
    const cards = document.querySelectorAll('[data-chart-name]');
    cards.forEach(async (card) => {
        const chartName = card.dataset.chartName;
        try {
            const versions = await fetchChartVersions(chartName);
            window.chartVersions[chartName] = versions;
        } catch (error) {
            console.error(`Error loading versions for ${chartName}:`, error);
        }
    });
});
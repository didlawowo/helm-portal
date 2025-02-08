// Main JavaScript file
console.log('Helm Portal initialized');
 
async function deleteChart(name, version) {
    try {
        const response = await fetch(`/chart/${name}/${version}`, {
            method: 'DELETE',
        });
        if (response.ok) {
            window.location.reload();
        } else {
            alert('Failed to delete chart');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error deleting chart');
    }
}
 
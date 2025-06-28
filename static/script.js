document.addEventListener('DOMContentLoaded', () => {
    const imageUpload = document.getElementById('imageUpload');
    const uploadButton = document.getElementById('uploadButton');
    const aiPrompt = document.getElementById('aiPrompt');
    const modifyButton = document.getElementById('modifyButton');
    const messageDiv = document.getElementById('message');
    const originalImage = document.getElementById('originalImage');
    const modifiedImage = document.getElementById('modifiedImage');

    let uploadedFileName = ''; // Store the name of the last uploaded file
    let uploadedAssetId = null; // Store the asset ID for AI modification

    function showMessage(msg, type = 'info') {
        messageDiv.textContent = msg;
        messageDiv.style.display = 'block';
        if (type === 'success') {
            messageDiv.style.backgroundColor = '#d4edda';
            messageDiv.style.color = '#155724';
        } else if (type === 'error') {
            messageDiv.style.backgroundColor = '#f8d7da';
            messageDiv.style.color = '#721c24';
        } else { // info
            messageDiv.style.backgroundColor = '#cce5ff';
            messageDiv.style.color = '#004085';
        }
    }

    // --- Image Upload Logic ---
    uploadButton.addEventListener('click', async () => {
        const file = imageUpload.files[0];
        if (!file) {
            showMessage('Please select an image file to upload.', 'error');
            return;
        }

        showMessage('Uploading image...', 'info');

        const formData = new FormData();
        formData.append('image', file); // 'image' must match the key in your Go handler (r.FormFile("image"))

        try {
            const response = await fetch('/assets', {
                method: 'POST',
                body: formData, // FormData automatically sets Content-Type: multipart/form-data
            });

            const data = await response.json(); // Assuming your Go server sends JSON response

            if (response.ok) {
                showMessage(data.message, 'success');
                uploadedAssetId = data.asset_id; // Store asset ID for later AI modification
                uploadedFileName = file.name; // Store filename for later AI modification
                originalImage.src = `/uploads/${uploadedFileName}`; // Assuming a future /uploads route
                originalImage.style.display = 'block'; // Show the original image
                modifiedImage.style.display = 'none'; // Hide modified until processed
            } else {
                showMessage(`Upload failed: ${data.message || response.statusText}`, 'error');
            }
        } catch (error) {
            console.error('Error during upload:', error);
            showMessage('An error occurred during upload. Check console.', 'error');
        }
    });

    // --- AI Modification Logic ---
    modifyButton.addEventListener('click', async () => {
    const prompt = aiPrompt.value.trim();
    if (uploadedAssetID === null) { // Check if an asset has been uploaded and ID received
        showMessage('Please upload an image first to get an Asset ID.', 'error');
        return;
    }
    if (!prompt) {
        showMessage('Please enter an AI modification prompt.', 'error');
        return;
    }

    showMessage('Applying AI modification...', 'info');
    modifiedImage.style.display = 'none';

    const requestBody = {
        asset_id: uploadedAssetId, // Use the stored asset ID
        prompt: prompt,
    };
});
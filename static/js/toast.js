//Show messages to the user
function showToast(message, type) {
    console.log("Displaying Toast.",message,type)
    var toastTypeClass = '';
    switch (type) {
        case 'error': toastTypeClass = 'bg-danger'; break;
        case 'warning': toastTypeClass = 'bg-warning text-dark'; break;
        case 'info': toastTypeClass = 'bg-success'; break;
        default: toastTypeClass = 'bg-secondary';
    }

    var toastId = 'toast' + Date.now(); // Unique ID for each toast
    var toastHtml = `
        <div class="toast ${toastTypeClass}" id="${toastId}" role="alert" aria-live="assertive" aria-atomic="true" data-bs-delay="5000">
            <div class="toast-body">
                ${message}
            </div>
        </div>`;

    $('#toastContainer').append(toastHtml);
    var toast = new bootstrap.Toast($('#' + toastId));
    toast.show();
    }
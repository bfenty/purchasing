
  function formatDateForDatePicker(dateString) {
    if (!dateString) return ""; // Handle null or undefined dates
    const date = new Date(dateString); // Convert ISO string to a Date object
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0'); // Month is zero-based
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`; // Return in YYYY-MM-DD format
}
    // Fetch users
    async function fetchUsers() {
        return new Promise((resolve, reject) => {
            $.ajax({
                url: '/api-handler?targetAPI=/api/users?limit=100&sorting=1&active=1', // Using the users API endpoint
                type: 'GET',
                dataType: 'json',
                success: function(response) {
                    console.log("Users fetched successfully. Data:", response);
                    // Assuming `response` includes a property `users` which is an array of user objects
                    const usernames = response.users.map(user => user.Username); // Extract usernames
                    resolve(usernames); // Resolve the promise with the array of usernames
                },
                error: function(jqXHR, textStatus, errorThrown) {
                    console.error("Error fetching users:", textStatus, errorThrown, jqXHR);
                    let errorMessage = "Error fetching users.";
                    
                    if (jqXHR.responseText) {
                        try {
                            let resp = JSON.parse(jqXHR.responseText);
                            if (resp && resp.error) {
                                errorMessage = resp.error;
                            }
                        } catch (e) {
                            console.error("Error parsing server response:", e);
                        }
                    }
                    reject(errorMessage); // Reject the promise with the error message
                }
            });
        });
    }
    async function fetchAndUpdateTable(currentPage, limit, searchParams = {}) {
        try {
            console.log("Fetching users...");
            const users = await fetchUsers(); // Fetch users
            console.log("Users fetched:", users);
    
            // Construct the query string from search parameters
            var limit = $('#limitSelect').val();
            var queryParams = $.param(searchParams); // Convert search parameters to query string
            var apiUrl = `/api-handler?targetAPI=/api/sortinglist&page=${currentPage}&limit=${limit}`;
    
            // Dynamically add conditions based on `layout`
            if (layout === "receiving") {
                apiUrl += "&search-checkin=NOT_NULL";
            } else if (layout === "checkout") {
                apiUrl += "&search-checkout=IS_NULL&search-status=New";
            } else if (layout === "checkin") {
                apiUrl += "&search-checkout=NOT_NULL&search-status=Checkout";
            }
    
            apiUrl += `&${queryParams}`;
    
            console.log("Fetching data from " + apiUrl);
    
            apicall(apiUrl, users);
        } catch (error) {
            console.error("Error in fetching users:", error);
            showToast(error, "error");
        }
    }
    

    function updateTable(response,users) {
        var tableBody = $('#contentTableBody');
        tableBody.empty(); // Clear existing data

        // Add the insert row at the beginning
        var insertRow = GenerateTopRow(users);
        tableBody.append(insertRow);

        // Loop through the rows
        response.sortRequests.forEach(function(sortRequest) {
        // Format the Checkin date for the date picker
        if (sortRequest.Checkin) {
            sortRequest.Checkin = formatDateForDatePicker(sortRequest.Checkin);
        }
        if (sortRequest.Checkout) {
            sortRequest.Checkout = formatDateForDatePicker(sortRequest.Checkout);
        }
            var row = generateTableRow(sortRequest,users);
            tableBody.append(row);
        });
    }

    function GenerateTopRow(users) {
      console.log("Generating Search Row");
      var userOptions = '<option value="" selected>Select Sorter</option>';
      
      users.forEach(function(user) {
          userOptions += `<option value="${user}">${user}</option>`;
      });

      // Layout conditions based on the Golang template value
      const showFull = layout === "full";
      const showReceiving = ["full", "receiving"].includes(layout);
      const showDataEntry = ["full", "dataentry"].includes(layout);
      const showMgmt = ["full", "mgmt"].includes(layout);

      // Generate the Search Row
      var rowHtml = `<tr>`;

      if (showFull || showReceiving || showDataEntry || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-requestid" placeholder="Request ID"></td>`;
      if (showFull || showReceiving || showDataEntry || showMgmt) 
          rowHtml += `<td class="col-sku"><input type="text" class="form-control" name="search-sku" placeholder="SKU"></td>`;
      if (showFull || showReceiving || showDataEntry) 
          rowHtml += `<td><input type="text" class="form-control" name="search-description" placeholder="Description"></td>`;
      if (showFull || showDataEntry) 
          rowHtml += `<td><input type="text" class="form-control" name="search-manufacturerpart" placeholder="Manufacturer Part"></td>`;
      if (showFull || showReceiving || showDataEntry || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-instructions" pattern="[0-9]+(\\.[0-9]+)?" title="Please enter a valid decimal number" placeholder="Pieces"></td>`;
      if (showFull || showReceiving || showDataEntry) 
          rowHtml += `<td><input type="text" class="form-control" name="search-weightout" placeholder="Weight Out"></td>`;
      if (showFull || showReceiving) 
          rowHtml += `<td><input type="text" class="form-control" name="search-weightin" placeholder="Weight In"></td>`;
      if (showFull || showReceiving) {
          rowHtml += `<td>
              <select class="form-select" name="search-difference">
                  <option value="">Difference</option>
                  <option value="positive">Positive</option>
                  <option value="negative">Negative</option>
              </select>
          </td>`;
      }
      if (showFull || showReceiving) rowHtml += `<td></td>`;
      if (showFull || showReceiving || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-pieces" pattern="[0-9]+(\\.[0-9]+)?" title="Please enter a valid decimal number" placeholder="Units"></td>`;
      if (showFull || showReceiving || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-hours" placeholder="Hours"></td>`;
      if (showFull || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-checkout" placeholder="Check Out"></td>`;
      if (showFull || showMgmt) 
          rowHtml += `<td><input type="text" class="form-control" name="search-checkin" placeholder="Check In"></td>`;
      if (showFull || showMgmt) {
          rowHtml += `<td>
              <select class="form-select" name="search-sorter">${userOptions}</select>
          </td>`;
      }
      if (showFull) {
          rowHtml += `<td>
              <select class="form-select" name="search-status">
                  <option value="">Status</option>
                  <option value="New">New</option>
                  <option value="Checkout">Checkout</option>
                  <option value="Checkin">Checkin</option>
              </select>
          </td>`;
      }
      if (showFull || showDataEntry) {
          rowHtml += `<td>
              <select class="form-select" name="search-priority">
                  <option value="">Priority</option>
                  <option value="0">Normal</option>
                  <option value="1">High</option>
              </select>
          </td>`;
      }
      if (showFull || showReceiving || showDataEntry || showMgmt) 
          rowHtml += `<td class="btn-col"><input type='submit' value='Search' class="btn btn-primary"></td>`;

      // Buttons    
      if (showFull || mgmt) rowHtml += `<td class="btn-col"></td>`;
      if (showFull || showReceiving || showMgmt) rowHtml += `<td class="btn-col"></td>`;
    

      rowHtml += `</tr>`;

      return rowHtml;
  }

  function generateTableRow(sortRequest, users) {

    // Strip percentage from percent
    let differencePercent = parseFloat(sortRequest.DifferencePercent.replace('%', ''));

    // Check if the absolute value is greater than 10
    let rowClass = '';
    if (Math.abs(differencePercent) > 20) {
        rowClass = 'table-danger';
    } else if (Math.abs(differencePercent) > 10) {
        rowClass = 'table-warning';
    }
    
    // Manufacturer dropdown options
    var userOptions = '<option value="" selected>Select Sorter</option>';
    users.forEach(function(user) {
        userOptions += `<option value="${user}" ${sortRequest.Sorter === user ? 'selected' : ''}>${user}</option>`;
    });

    // Layout conditions
    const showFull = layout === "full";
    const showReceiving = ["full", "receiving"].includes(layout);
    const showDataEntry = ["full", "dataentry"].includes(layout);
    const showMgmt = ["full", "mgmt"].includes(layout);
    const showCheckout = ["full", "checkout"].includes(layout);
    const showCheckin = ["full", "checkin"].includes(layout);
    
    // Constructing the row HTML
    var rowHtml = `<tr class="${rowClass}">`;
    rowHtml += `<td><input type="text" class="form-control requestid" id="requestid" name="requestid" value="${sortRequest.ID}" readonly></td>`;
    if (showFull || showReceiving || showDataEntry || showMgmt || showCheckout || showCheckin) rowHtml += `<td class="col-sku"><input type="text" class="form-control sku" id="sku" name="sku" value="${sortRequest.SKU}" ${!mgmt ? 'readonly' : ''}></td>`;
    if (showFull || showReceiving || showDataEntry || showCheckout || showCheckin) rowHtml += `<td><input type="text" class="form-control description" id="description" name="description" value="${sortRequest.Description}" ${!mgmt ? 'readonly' : ''}></td>`;
    if (showFull || showDataEntry) rowHtml += `<td><input type="text" class="form-control manufacturerpart" id="manufacturerpart" name="manufacturerpart" value="${sortRequest.ManufacturerPart}"></td>`;
    if (showFull || showReceiving || showDataEntry || showMgmt || showCheckout || showCheckin) rowHtml += `<td><input type="text" class="form-control instructions" id="instructions" name="instructions" pattern="[0-9]+(\.[0-9]+)?" title="Please enter a valid decimal number" value="${sortRequest.Instructions}" ${!mgmt ? 'readonly' : ''}></td>`;
    if (showFull || showReceiving || showDataEntry) rowHtml += `<td><input type="text" class="form-control weightout" id="weightout" name="weightout" value="${sortRequest.Weightout}"></td>`;
    if (showFull || showReceiving) rowHtml += `<td><input type="text" class="form-control weightin" id="weightin" name="weightin" value="${sortRequest.Weightin}"></td>`;
    if (showFull || showReceiving) rowHtml += `<td><input type="text" class="form-control difference" id="difference" name="difference" value="${sortRequest.Difference}" readonly></td>`;
    if (showFull || showReceiving) rowHtml += `<td><input type="text" class="form-control differencepercent" id="differencepercent" name="differencepercent" value="${sortRequest.DifferencePercent}" readonly></td>`;
    if (showFull || showReceiving || showMgmt || showCheckin) rowHtml += `<td><input type="text" class="form-control pieces" id="pieces" name="pieces" value="${sortRequest.Pieces}"></td>`;
    if (showFull || showReceiving || showMgmt || showCheckin) rowHtml += `<td><input type="text" class="form-control hours" id="hours" name="hours" value="${sortRequest.Hours}"></td>`;
    if (showFull || showMgmt || showCheckout || showCheckin) rowHtml += `<td><input type="date" class="form-control checkout" id="checkout" name="checkout" value="${sortRequest.Checkout || new Date().toISOString().split('T')[0]}" ${!mgmt ? 'readonly' : ''}></td>`;
    if (showFull || showMgmt || showCheckin) rowHtml += `<td><input type="date" class="form-control checkin" id="checkin" name="checkin" value="${sortRequest.Checkin || new Date().toISOString().split('T')[0]}" ${!mgmt ? 'readonly' : ''}></td>`;
    if (showFull || showMgmt || showCheckout || showCheckin) rowHtml += `<td><select class="form-select" id="sorter" name="sorter">${userOptions}</select></td>`;
    if (showFull) {
        rowHtml += `<td>`;
        if (admin) {
            rowHtml += `<select class="form-select" id="status" name="status">
                <option value="New" ${sortRequest.Status === "New" ? 'selected' : ''}>New</option>
                <option value="Checkout" ${sortRequest.Status === "Checkout" ? 'selected' : ''}>Checkout</option>
                <option value="Checkin" ${sortRequest.Status === "Checkin" ? 'selected' : ''}>Checkin</option>
            </select>`;
        } else {
            rowHtml += `<input class="form-control" type="text" id="status" name="status" value="${sortRequest.Status}" readonly>`;
        }
        rowHtml += `</td>`;
    }
    if (showFull || showDataEntry) rowHtml += `<td><select class="form-select" id="prty" name="prty"><option value="0">Normal</option><option value="1">High</option></select></td>`;

    // Buttons
    if (showFull || showReceiving || showDataEntry || showMgmt) rowHtml += `<td class="btn-col"><input type='hidden' id='active' name='active' value='1'><input type='submit' value='Update' class="btn btn-primary update-request"></td>`;
    if (showFull || mgmt) rowHtml += `<td class="btn-col"><button type="button" class="btn btn-danger archive-request" id="archiveRequest" name="archiveRequest" data-requestid="${sortRequest.ID}">Archive</button></td>`;
    if (showFull || showReceiving || showMgmt) rowHtml += `<td class="btn-col"><a href="/sorterror?requestid=${sortRequest.ID}" class="btn btn-warning error" id="error" name="error" data-requestid="">Error</a></td>`;
    
    rowHtml += `</tr>`;
    return rowHtml;
}
    // Event listener for the archive button
      $('#contentTableBody').on('click', '.archive-request', function () {
          var requestID = $(this).data('requestid'); // Retrieve the request ID from the button's data attribute
          console.log("Opening archive modal for request ID:", requestID);

          // Update the modal with the request ID
          $('#archiveRequestID').text(requestID);
          $('#confirmArchiveRequest').data('requestid', requestID); // Pass the request ID to the confirm button

          // Show the modal
          $('#archiveRequestModal').modal('show');
      });

      $(document).on('click', '#confirmArchiveRequest', function () {
      var requestID = $(this).data('requestid'); // Retrieve the request ID from the button's data attribute
      console.log("Archiving request ID using sortingupdate API:", requestID);

      if (!requestID) {
          console.error("No request ID found for archiving");
          showToast("Error: No request ID found", "error");
          return;
      }

      // Declare and define the payload object
      var payload = {
          ID: requestID,
          Active: 0 // Mark as archived
      };

      // AJAX call to update the request
      $.ajax({
          url: '/api-handler?targetAPI=/api/sortingupdate', // Use the sortingupdate API endpoint
          type: 'POST',
          contentType: 'application/json',
          data: JSON.stringify(payload),
          success: function (response) {
              console.log("Request archived successfully:", response);
              showToast("Request archived successfully", "success");
              $('#archiveRequestModal').modal('hide'); // Close the modal
              fetchAndUpdateTable(); // Refresh the table
          },
          error: function (jqXHR, textStatus, errorThrown) {
              console.error("Error archiving request:", textStatus, errorThrown);
              var errorMessage = "Error archiving request.";
              if (jqXHR.responseText) {
                  try {
                      var resp = JSON.parse(jqXHR.responseText);
                      if (resp && resp.error) {
                          errorMessage = resp.error;
                      }
                  } catch (e) {
                      console.error("Error parsing server response:", e);
                  }
              }
              showToast(errorMessage, "error");
          }
      });
  });


    // Event listener for the update button click
    $('#contentTableBody').on('click', '.update-request', function () {
        var row = $(this).closest('tr'); // Find the closest row
        var requestID = row.find('[name="requestid"]').val();

        // Collect and parse data from the row
        var updatedRequestData = {
            ID: parseInt(requestID, 10) || 0,
            SKU: row.find('[name="sku"]').val(),
            Description: row.find('[name="description"]').val(),
            ManufacturerPart: row.find('[name="manufacturerpart"]').val(),
            Instructions: row.find('[name="instructions"]').val(),
            WeightIn: parseFloat(row.find('[name="weightin"]').val()) || 0,
            WeightOut: parseFloat(row.find('[name="weightout"]').val()) || 0,
            Pieces: parseInt(row.find('[name="pieces"]').val(), 10) || 0,
            Hours: parseFloat(row.find('[name="hours"]').val()) || 0,
            Sorter: row.find('[name="sorter"]').val(),
            Status: row.find('[name="status"]').val(),
            Priority: parseInt(row.find('[name="priority"]').val(), 10) || 0,
        };

        console.log("Preparing to update sort request:", updatedRequestData);

        // AJAX call to update the sort request
        $.ajax({
            url: '/api-handler?targetAPI=/api/sortingupdate',
            type: 'POST',
            contentType: 'application/json',
            data: JSON.stringify(updatedRequestData),
            success: function (response) {
                console.log("Sort request updated successfully:", response);
                showToast("Sort request updated successfully.", "info");
                fetchAndUpdateTable(currentPage, limit, searchParams); // Refresh the table
            },
            error: function (jqXHR, textStatus, errorThrown) {
            console.error("Error during the request:", textStatus, errorThrown, jqXHR);

            // Handle errors and provide detailed feedback
            var errorMessage = "An unexpected error occurred.";
            if (jqXHR.responseText) {
                try {
                    // Attempt to parse the API response
                    var resp = JSON.parse(jqXHR.responseText);
                    if (resp && resp.error) {
                        errorMessage = resp.error; // Use the error message from the API
                    }
                } catch (e) {
                    console.error("Error parsing server response:", e);
                    errorMessage = "Failed to parse server response.";
                }
            }
            showToast(errorMessage, "error"); // Display error message in the toast
        }
        });
    });

    document.addEventListener("DOMContentLoaded", function () {
        console.log("DOM fully loaded. Waiting for search button...");
    
        // Delegate event listener for the dynamically added Search button
        document.getElementById("contentTableBody").addEventListener("click", function (event) {
            if (event.target.type === "submit" && event.target.value === "Search") {
                event.preventDefault(); // Prevent default form submission
                console.log("Search button clicked. Collecting search parameters...");
    
                // Collect search input values
                let searchParams = {};
                document.querySelectorAll("[name^='search-']").forEach(input => {
                    if (input.value.trim() !== "") {
                        searchParams[input.name] = input.value.trim();
                    }
                });
    
                console.log("Search parameters:", searchParams);
    
                // Fetch and update table with search parameters
                fetchAndUpdateTable(1, $('#limitSelect').val(), searchParams);
            }
        });
    });
    
//Not yet used, need to migrate from using scripts.html

function GenerateTopRow(manufacturers, currencies, seasons) {
    console.log("Generating Search Row");

    // Manufacturer dropdown options
    var manufacturerOptions = '<option value="" selected>Select Manufacturer</option>';
    manufacturers.forEach(function(manufacturer) {
        manufacturerOptions += `<option value="${manufacturer.name}">${manufacturer.name}</option>`;
    });

    // Generate Currency Options
    var currencyOptions = '<option value="" selected>Select Currency</option>';
    currencies.forEach(function(currency) {
        currencyOptions += `<option value="${currency}">${currency}</option>`;
    });

    // Generate Seasons Options
    var seasonOptions = '<option value="" selected>None</option>';
    seasons.forEach(function(season) {
        seasonOptions += `<option value="${season}">${season}</option>`;
    });

    // Start building the row
    var insertRow = `<tr class="table-danger">
        <td></td>
        <td><input class="form-control" type='text' id="sku"></td>
        <td><input class="form-control" type='text' id="manufacturerpart"></td>
        <td><input class="form-control" type='text' id="description"></td>`;

    // Only add admin fields if user is an admin
    if (permission === "admin") {
        insertRow += `<td><select class="form-select" id="manufacturer">${manufacturerOptions}</select></td>`;
        insertRow += `<td><input class="form-control" type='text' id="processrequest"></td>`;
    }

    insertRow += `<td><input class="form-control" type='text' id="unit"></td>`;

    if (permission === "admin") {
        insertRow += `<td><input class="form-control" type='text' id="unitprice"></td>`;
        insertRow += `<td><select class="form-select" id="currency">${currencyOptions}</select></td>`;
        insertRow += `<td><input class="form-control" type='text' id="orderqty"></td>`;
        insertRow += `<td><input class="form-check-input" type='checkbox' id="reorder" name="reorder" checked></td>`;
    }

    insertRow += `<td><select class="form-select" id="season">${seasonOptions}</select></td>
        <td><input class="form-control" type='text' id="new_inventoryqty"></td>
        <td><button class="btn btn-primary" id="btnSearch">Search</button></td>
        <td></td>
        <td></td>
    </tr>`;

    return insertRow;
}

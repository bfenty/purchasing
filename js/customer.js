

{{define "searchCustomerRow"}}
<script>
    function GenerateTopRow() {
        console.log("Generating Search Row")

        // Generate the Insert Row
        var insertRow = `
        <tr class="search-row">
            <td><input class="form-control" type="text" id="customer_email" ></td>
            <td><input class="form-control" type="text" id="first_name" </td>
            <td><input class="form-control" type="text" id="last_name" ></td>
            <td><input class="form-control" type="text" id="country" ></td>
            <td><input class="form-control" type="text" id="rebill_day" ></td>
            <td><input class="form-control" type="text" id="rebill_months" ></td>
            <td><input class="form-control" type="text" id="autorenew" ></td>
            <td><input class="form-control" type="text" id="cratejoy_status" ></td>
            <td><input class="form-control" type="text" id="start_date" ></td>
            <td><input class="form-control" type="text" id="end_date" ></td>
            <td><input class="form-control" type="text" id="mailchimp_status"></td>
            <td><button class="btn btn-primary" id="btnSearch">Search</button></td>
            <td></td>
        </tr>
        `;

        // Returning the row HTML
        return insertRow;
    }
</script>
{{end}}

function getResponse(path, onResult) {
    var request = new XMLHttpRequest();
    request.open('GET', '/categories', true);
    request.setRequestHeader("Content-Type", "application/json");
    request.onload = function() {
        var resp = this.response;
        onResult(JSON.parse(resp));
    };
    request.send();
}

getResponse("/categories", function(categories) {
    var select = document.querySelector("#categories");
    for (var i = 0; i < categories.length; i++) {
        select.innerHTML += "<option>"+categories[i]+"</option>";
    }
});
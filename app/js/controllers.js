var app = angular.module("aroundtheworld-app", []);

app.controller("maincontroller", [ "$scope", "$http", function($scope, $http) {
  $scope.victimCount = 0;
  var originalCount = 0;
  $scope.submitForm = function(user) {
    $("#email").css("border-bottom-color", "#3da9df")
    $http.post("/v1/email", user).then(function successCallback(response) {
      user.Email = "";
      user.ZIP = "";
    }, function errorCallback(response) {
      console.log(response);
      $("#email").css("border-bottom-color", "red")
    });
  };

  $scope.getCount = function() {
    $http.get("/v1/victimCount").then(function successCallback(response) {
      originalCount = parseInt(response.data);
      $scope.victimCount = originalCount;
      $("#hero-wrapper").addClass("slide-down");
    }, function errorCallback(response) {
      $scope.victimCount = "error";
      $("#hero-wrapper").addClass("slide-down");
    });
  };

  $scope.getCount();
}]);

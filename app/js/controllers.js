var app = angular.module("aroundtheworld-app", []);

app.controller("maincontroller", [ "$scope", "$http", function($scope, $http) {
  $scope.victimCount = "0";
  $scope.submitForm = function(user) {
    $("#email").css("border-bottom-color", "#3da9df")
    $http.post("/email", user).then(function successCallback(response) {
      user.Email = "";
      user.State = "";
    }, function errorCallback(response) {
      $("#email").css("border-bottom-color", "red")
    });
  };

  $scope.getCount = function() {
    $http.get("/victimCount").then(function successCallback(response) {
      $scope.victimCount = response.data;
    }, function errorCallback(response) {
      $scope.victimCount = "error";
    });
  };

  $scope.getCount();
}]);

<template>
  <div id="container">
    <el-form ref="form" label-width="120px" >
      <el-form-item label="账号">
        <el-input v-model="form.Username"></el-input>
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.Password" type="password"></el-input>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="admin">登陆</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script>
    export default {
        name: "admin",
      data(){
        return {
          objects:"",
          way:"",
          form:{
            Username : "",
            Password : "",
          },
          formjs:[{
            Username : "",
            Password : "",
            CompanyName : "",
            CompanyId : "",
          }],
        }
        },
    methods:{

      admin(){
        // for simple empty check
        // According to admin infos to jump to different user interface
        if(this.form.Username !== "" && this.form.Password !== ""){
            if(this.form.Username === "admin"){
                this.axios.post('http://115.28.136.131:8000/A1/admin/admintest', this.form)
                  .then(function (res) {
                    this.formjs = JSON.parse(res.data);
                    this.$router.push({name:"bankInterface", params:{CompanyName:this.formjs.CompanyName,CompanyId:this.formjs.CompanyId}, requiredAuth:true});
                  }.bind(this))
                  .catch(function (err) {
                    alert("请填写正确信息!")
                    this.form.Username=""
                    this.form.Password=""
                    if (err.response){
                      //失败
                      console.log(err.response)
                      }
                  }.bind(this))
            }else{
                this.axios.post('http://115.28.136.131:8000/A1/admin/admintest', this.form)
                  .then(function (res) {
                    this.formjs = JSON.parse(res.data);
                    this.$router.push({name:"companyInterface", params:{CompanyName:this.formjs.CompanyName,CompanyId:this.formjs.CompanyId}, requiredAuth:true});
                  }.bind(this))
                  .catch(function (err) {
                    alert("请填写正确信息!")
                    this.form.Username=""
                    this.form.Password=""
                    if (err.response){
                      //失败
                      console.log(err.response)
                    }
                  }.bind(this))
            }
        }else{
          alert("请填写完整信息!")
        }
      }
    }
    }
</script>

<style scoped>
  #container{
    width: 600px;
    height: 1500px;
    margin: 200px auto;

  }
</style>

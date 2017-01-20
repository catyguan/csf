# CSF Console控制程序

## 使用方法

root := csfctl.NewRootEnv()
root.CreateStandardCommands() // 添加标准命令集合
rdir := root.RootDir()
if true { // 添加应用命令集合
	dir := rdir.CreateDir("counter")
	dir.AddCommand(createGETCommand())
	dir.AddCommand(createADDCommand())
}
root.RunAsConsole() // 启动
os.Exit(1)

## 目录扩展方法

新加目录
	CommandDir.AddDir("name")
建立或获取目录
	CommandDir.CreateDir("name")
新加命令
	dir.AddCommand(cmd)

## 命令扩展方法

### 实现命令处理接口

type CommandAction func(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error
	env -- 命令工作环境
	pwd -- 命令所在目录
	cmdobj -- 命令定义数据
	args -- 执行参数

func handleExitCommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, _ *csfctl.Command, args []string) error {
	env.Printf("exit ENV[%p]\n", env)
	env.PopEnv()
	return nil
}

### 定义命令

&Command{
	Name:        "exec",
	Usage:       "exec {filename}",
	Description: `read a file and process command line by line`,
	Aliases:     []string{},
	Args: Flags{
		Flag{Name: "i", Type: "bool", Usage: "ignore error"},
		Flag{Name: "n", Type: "bool", Usage: "process command in new Env"},
		Flag{Name: "h", Type: "bool", Usage: "show help"},
	},
	Vars: Flags{
		Flag{Name: "FWD", Type: "string", Usage: "File system Work Directory"},
	},
	Action: HandleEXECCommand,
}

## 技巧

### 处理执行参数

vars, nargs, err := cmdobj.Args.Parse(args)
if err != nil {
	return env.PrintErrorf(err.Error())
}
args = nargs

varH := vars["h"].(bool)
varN := vars["n"].(bool)
varI := vars["i"].(bool)

支持的参数类型：
"bool"->bool, "string"->string, "int"->int, "int64"->int64,"float"->float64,"duration"->time.Duration

### 处理环境变量

env.GetVar(name, defaultValue)
env.GetVarString(...)
env.GetVarBool(...)
env.GetVarXXXX

环境变量具有传递特性，子环境会读取父环境的变量

### 命令行引用环境变量

设置命令行非严格模式
env set STRICT false
控制变量STRICT缺省值为false

引用
echo $STRICT
echo ${STRICT}

代码中引用
env.FormatVars(s) -- 会判断STRICT设置
env.DoFormatVars(s) -- 直接处理

### 显示帮助

csfctl.DoHelp(ctx, env, cmdobj)

### 命令定位

csfctl.LocateCommand(pwd, cmdStr)

### 输出和错误处理

env.Printf
env.Prinfxxx
return env.PrintErrorf
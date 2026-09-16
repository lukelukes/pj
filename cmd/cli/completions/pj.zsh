_pj_projects() {
    local projects=(${(f)"$(pj list -n 2>/dev/null)"})
    compadd -S '' -- $projects
}

_pj_filter_fields() {
    compadd -S '' -- name= path= editor= desc=
}

_pj() {
    local -a commands=(
        'a:Add a project to the catalog'
        'add:Add a project to the catalog'
        'ls:List projects in the catalog'
        'list:List projects in the catalog'
        'rm:Remove a project from the catalog'
        'o:Open project in editor'
        'open:Open project in editor'
        'e:Edit project metadata'
        'edit:Edit project metadata'
        'create:Create a new project'
        'new:Create a new project'
        'show:Show project details'
        'cd:Change directory to project'
        'init:Generate shell integration'
        'completion:Generate shell completions'
    )

    _arguments -C \
        '(-h --help)'{-h,--help}'[Show help]' \
        '(-c --catalog)'{-c,--catalog}'[Path to catalog file]:path:_files' \
        '1:command:->cmds' \
        '*::arg:->args'

    case $state in
        cmds) _describe 'command' commands ;;
        args)
            case $line[1] in
                a|add)
                    _arguments \
                        '(-n --name)'{-n,--name}'[Project name]:name:' \
                        '1:path:_files -/'
                    ;;
                ls|list)
                    _arguments \
                        '*'{-f,--filter}'[Filter projects]:filter:_pj_filter_fields' \
                        '(-o --output)'{-o,--output}'[Output format]:format:(table names paths json)'
                    ;;
                rm)
                    _arguments '1:project:_pj_projects'
                    ;;
                o|open)
                    _arguments '1:project:_pj_projects'
                    ;;
                e|edit)
                    _arguments \
                        '--desc[Set description]:description:' \
                        '--editor[Set editor]:editor:' \
                        '1:project:_pj_projects'
                    ;;
                create|new)
                    _arguments \
                        '--at[Parent directory]:directory:_files -/' \
                        '--desc[Project description]:description:' \
                        '--editor[Editor command]:editor:' \
                        '--no-git[Skip git initialization]' \
                        '--adopt[Adopt an existing directory]' \
                        '--no-input[Never prompt]' \
                        '1:name:'
                    ;;
                show)
                    _arguments \
                        '--path[Output only the path]' \
                        '1:project:_pj_projects'
                    ;;
                cd)
                    _arguments '1:project:_pj_projects'
                    ;;
                completion)
                    _arguments '1:shell:(zsh)'
                    ;;
            esac
            ;;
    esac
}

compdef _pj pj

_pj_projects() {
    local projects=(${(f)"$(pj list -n 2>/dev/null)"})
    compadd -S '' -- $projects
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
        's:Search for projects'
        'search:Search for projects'
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
                        '(-t --tags)'{-t,--tags}'[Tags]:tags:' \
                        '1:path:_files -/'
                    ;;
                ls|list)
                    _arguments \
                        '(-s --status)'{-s,--status}'[Filter by status]:status:(active archived abandoned)' \
                        '(-T --types)'{-T,--types}'[Filter by types]:types:' \
                        '(-t --tags)'{-t,--tags}'[Filter by tags]:tags:' \
                        '(-r --recent)'{-r,--recent}'[Sort by last accessed]' \
                        '--json[Output as JSON]' \
                        '(-n --names)'{-n,--names}'[Output only names]'
                    ;;
                rm)
                    _arguments '1:project:_pj_projects'
                    ;;
                o|open)
                    _arguments '1:project:_pj_projects'
                    ;;
                e|edit)
                    _arguments \
                        '(-s --status)'{-s,--status}'[Set status]:status:(active archived abandoned)' \
                        '--add-tag[Add tags]:tag:' \
                        '--rm-tag[Remove tags]:tag:' \
                        '--notes[Set notes]:notes:' \
                        '--editor[Set editor]:editor:' \
                        '1:project:_pj_projects'
                    ;;
                s|search)
                    _arguments '1:query:'
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
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
            esac
            ;;
    esac
}

compdef _pj pj

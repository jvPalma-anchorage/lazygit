_This file is auto-generated. To update, make the changes in the pkg/i18n directory and then run `go generate ./...` from the project root._

# Lazygit Atalhos do teclado

## Combinações globais de teclas

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+r> `` | Mudar para um repositório recente |  |
| `` <pgup>, K, <ctrl+u> (fn+up/shift+k) `` | Rolar janela principal para cima |  |
| `` <pgdown>, J, <ctrl+d> (fn+down/shift+j) `` | Rolar a janela principal para baixo |  |
| `` @ `` | Ver opções do registo de comandos | Ver opções do registo de comandos, p. ex. mostrar/esconder o registo de comandos e focá-lo. |
| `` P `` | Empurre (Push) | Faz push do branch atual para o seu branch upstream. Se não houver upstream configurado, é-te pedido que configures um branch upstream. Se houver outros branches empilhados abaixo do atual com commits por enviar, é-te proposto fazer push deles também. |
| `` p `` | Puxar (Pull) | Faz pull das alterações do remote para o branch atual. Se não houver upstream configurado, é-te pedido que configures um branch upstream. Se houver outros branches empilhados abaixo do atual que mudaram no remote, é-te proposto atualizá-los também. |
| `` ) `` | Aumentar o limiar de semelhança para renomeações | Aumenta o limiar de semelhança a partir do qual um par remoção/adição é tratado como renomeação.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'git.renameSimilarityThreshold'. |
| `` ( `` | Diminuir o limiar de semelhança para renomeações | Diminui o limiar de semelhança a partir do qual um par remoção/adição é tratado como renomeação.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'git.renameSimilarityThreshold'. |
| `` } `` | Aumentar o contexto do diff | Aumenta a quantidade de contexto mostrada à volta das alterações na vista de diff.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'git.diffContextSize'. |
| `` { `` | Diminuir o contexto do diff | Diminui a quantidade de contexto mostrada à volta das alterações na vista de diff.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'git.diffContextSize'. |
| `` : `` | Executar comando da shell | Traga um prompt onde você pode digitar um comando shell para executar. |
| `` <ctrl+p> `` | Ver opções de patch personalizadas |  |
| `` m `` | Ver opções de mesclar/rebase | Ver opções para abortar/continuar/pular o merge/rebase atual. |
| `` R `` | Atualizar | Atualize o estado do git (ou seja, execute `git status`, `git branch`, etc em segundo plano para atualizar o conteúdo de painéis). Isso não executa `git fetch`. |
| `` + `` | Modo de tela seguinte (normal/metade/tela cheia) |  |
| `` _ `` | Modo de tela anterior |  |
| `` \| `` | Alternar renderizadores de diff | Escolhe o renderizador seguinte na lista de renderizadores de diff configurados. |
| `` \ `` | Alternar renderizadores de diff (inverso) | Escolhe o renderizador anterior na lista de renderizadores de diff configurados. |
| `` <ctrl+g> `` | Saltar para ficheiro no diff | Escolhe um dos ficheiros do diff mostrado na vista principal e desloca a vista principal até ele. O foco mantém-se neste painel. |
| `` <esc> `` | Cancelar |  |
| `` ? `` | Abrir o menu de atalhos do teclado |  |
| `` <ctrl+s> `` | Ver opções de filtro | Ver opções para filtrar o log de commits, de modo a mostrar só os commits que correspondem ao filtro. |
| `` W, <ctrl+e> `` | Ver opções de diff | Ver opções relativas a comparar duas refs, p. ex. comparar com a ref selecionada, introduzir a ref a comparar e inverter a direção do diff. |
| `` q, <ctrl+c> `` | Sair |  |
| `` <ctrl+z> `` | Suspender a aplicação |  |
| `` <ctrl+w> `` | Alternar espaços em branco | Alterna entre mostrar ou não as alterações de espaços em branco na vista de diff.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'git.ignoreWhitespaceInDiffView'. |
| `` <alt+shift+c> `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` z `` | Desfazer | O reflog será usado para determinar qual comando git para executar para desfazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |
| `` Z `` | Refazer | O reflog será usado para determinar qual comando git para executar para refazer o último comando git. Isto não inclui mudanças na árvore de trabalho; apenas compromissos são tidos em consideração. |

## Navegação nos painéis de lista

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` , `` | Aba anterior |  |
| `` . `` | Próxima aba |  |
| `` <, <home> `` | Voltar ao topo |  |
| `` >, <end> `` | Ir para o final |  |
| `` v `` | Alternar seleção de intervalo |  |
| `` <shift+down> `` | Selecionar intervalo para baixo |  |
| `` <shift+up> `` | Selecionar intervalo para cima |  |
| `` / `` | Pesquisar na visualização atual por texto |  |
| `` H `` | Rolar à esquerda |  |
| `` L `` | Scroll para a direita |  |
| `` ] `` | Próxima aba |  |
| `` [ `` | Aba anterior |  |

## Arquivos

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar caminho para área de transferência |  |
| `` <space> `` | Etapa | Alternar para staging para o arquivo selecionado. |
| `` <ctrl+b> `` | Filtrar arquivos por status |  |
| `` y `` | Copiar para a área de transferência |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` w `` | Fazer commit de alterações sem pré-commit |  |
| `` A `` | Alterar último commit |  |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` i `` | Ignore or exclude file |  |
| `` r `` | Atualizar arquivos |  |
| `` s `` | Stash | Stash todas as alterações. Para outras variações de armazenamento, use a fixação de teclas de armazenamento. |
| `` S `` | Ver opções de stash | Ver opções de stash (por exemplo, trash all, stash staged, stash unsttued). |
| `` a `` | Stage completo | Alternar para todos os arquivos na árvore de trabalho |
| `` <enter> `` | Focar diff do ficheiro / Recolher diretório | Se o item selecionado for um ficheiro, foca o seu diff para poderes agir sobre hunks ou linhas individuais. Se for um diretório, recolhe-o ou expande-o. |
| `` d `` | Descartar | Exibir opções para descartar alterações para o arquivo selecionado. |
| `` g `` | Ver opções de reset para o upstream |  |
| `` D `` | Restaurar | Opções de redefinição de exibição para árvore de trabalho (por exemplo, nukando a árvore de trabalho). |
| `` ` `` | Alternar exibição de árvore de arquivo | Alterna a vista de ficheiros entre lista e árvore. A lista mostra todos os caminhos de ficheiros numa só lista; a árvore agrupa os ficheiros por diretório.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'gui.showFileTree'. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` M `` | Ver opções de conflitos de merge | Ver opções para resolver conflitos de merge. |
| `` f `` | Buscar | Buscar alterações do controle remoto. |
| `` - `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` = `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches locais

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar nome da branch para área de transferência |  |
| `` i `` | Exibir opções do git-flow |  |
| `` <space> `` | Verificar | Checar item selecionado |
| `` n `` | Nova branch |  |
| `` N `` | Mover commits para uma nova branch | Cria um novo branch e move para lá os commits do branch atual que ainda não foram enviados (push). Útil quando querias começar trabalho novo e te esqueceste de criar primeiro um branch.<br><br>Isto ignora a seleção: o novo branch é sempre criado a partir do branch principal ou empilhado sobre o branch atual (podes escolher). |
| `` w `` | Nova árvore de trabalho |  |
| `` o `` | Criar solicitação de pull |  |
| `` O `` | Ver opções para criar pull request |  |
| `` G `` | Abrir pull request no browser |  |
| `` <ctrl+y> `` | Copiar URL do pull request para área de transferência |  |
| `` c `` | Checar por nome | Checar por nome. Na caixa de entrada você pode inserir '-' para trocar para a última branch  |
| `` - `` | Checkout da branch anterior |  |
| `` F `` | Forçar checagem | Forçar checagem da branch selecionada. Isso irá descartar todas as mudanças no seu diretório de trabalho antes cheque a branch selecionada   |
| `` d `` | Apagar | Ver opções de exclusão para a branch local/remoto. |
| `` r `` | Refazer | Refazer a branch checada na branch selecionada |
| `` M `` | Mesclar | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` f `` | Avanço rápido | Faz fast-forward do branch selecionado a partir do seu upstream. Se o branch divergiu do upstream porque o branch upstream foi reescrito, e não tem commits próprios, é reposto para o upstream. Isto requer os reflogs ativos; um repositório bare não os guarda por omissão (core.logAllRefUpdates). |
| `` T `` | Nova etiqueta |  |
| `` s `` | Ordenação |  |
| `` g `` | Restaurar |  |
| `` R `` | Renomear branch |  |
| `` u `` | Ver opções de upstream | Ver opções relativas ao upstream do branch, p. ex. definir/remover o upstream e repor para o upstream. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Branches remotos

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar nome da branch para área de transferência |  |
| `` <space> `` | Verificar | Checar a nova branch baseada na brach remota selecionada, ou a branch remota como HEAD, desanexado |
| `` n `` | Nova branch |  |
| `` w `` | Nova árvore de trabalho |  |
| `` M `` | Mesclar | Ver opções para mesclar o item selecionado no branch atual (mesclar regularmente, mesclar squash) |
| `` r `` | Refazer | Refazer a branch checada na branch selecionada |
| `` d `` | Apagar | Excluir o branch remoto do controle remoto. |
| `` u `` | Definir como upstream | Definir o ramo remoto selecionado como fluxo do branch check-out. |
| `` s `` | Ordenação |  |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Checks

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Expandir/recolher workflow ou ver logs do check |  |

## Commit arquivos

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar caminho para área de transferência |  |
| `` y `` | Copiar para a área de transferência |  |
| `` c `` | Verificar | Arquivo de check-out. Isso substitui o arquivo em sua árvore de trabalho com a versão do commit selecionado. |
| `` d `` | Descartar | Descartar as alterações desse commit para este arquivo. Isso executa uma rebase interativa em segundo plano, então você pode ter um conflito de merge se um commit posterior também alterar este arquivo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` e `` | Editar | Abrir arquivo no editor externo. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` <space> `` | Alternar entre o arquivo incluído no patch | Alternar se o arquivo está incluído no patch personalizado. Veja https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` a `` | Alternar todos os arquivos | Adicionar/remover todos os arquivos de commit para atualização personalizada. Consulte https://github.com/jesseduffield/lazygit#rebase-magic-custom-patches. |
| `` <enter> `` | Focar diff do ficheiro / Alternar diretório | Se estiver selecionado um ficheiro, foca o seu diff para poderes agir sobre linhas individuais. Se for um diretório, recolhe-o ou expande-o. |
| `` ` `` | Alternar exibição de árvore de arquivo | Alterna a vista de ficheiros entre lista e árvore. A lista mostra todos os caminhos de ficheiros numa só lista; a árvore agrupa os ficheiros por diretório.<br><br>O valor por omissão pode ser alterado no ficheiro de configuração com a chave 'gui.showFileTree'. |
| `` - `` | Recolher todos os arquivos | Recolher todos os diretórios na árvore de arquivos |
| `` = `` | Expandir todos os arquivos | Expandir todos os diretórios na árvore do arquivo |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Commits

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar hash abreviado do commit para a área de transferência |  |
| `` <ctrl+r> `` | Limpar a seleção de commits copiados (cherry-pick) |  |
| `` <enter> `` | Focar as alterações do commit na vista principal |  |
| `` b `` | Ver opções de bissecção |  |
| `` s `` | Squash | Squash o commit selecionado no commit abaixo dele. A mensagem do commit selecionado será anexada ao commit abaixo dele. |
| `` f `` | Corrigir | Faça o commit selecionado no commit abaixo dele. Semelhante para o squash, mas a mensagem do commit selecionado será descartada. |
| `` c `` | Configurar mensagem de correção | Defina a opção de mensagem para o commit de correção. A opção -C significa usar a mensagem deste commit em vez da mensagem do commit alvo. |
| `` r `` | Reword | Repetir a mensagem de submissão selecionada. |
| `` R `` | Republicar com o editor |  |
| `` d `` | Descartar | Solte o commit selecionado. Isso irá remover o commit do branch através de uma rebase. Se o commit faz com que as alterações em commits posteriores dependem, você pode precisar resolver conflitos de merge. |
| `` e `` | Editar (iniciar rebase interativa) | Editar o commit selecionado. Use isto para iniciar uma rebase interativa a partir do commit selecionado. Quando já estiver no meio da reconstrução, isto irá marcar o commit selecionado para edição, o que significa que ao continuar com a reformulação. a rebase irá pausar no commit selecionado para permitir que você faça alterações. |
| `` i `` | Iniciar rebase interativo | Inicia um rebase interativo dos commits do teu branch. Inclui todos os commits desde o commit HEAD até ao primeiro merge commit ou commit do branch principal.<br>Se preferires iniciar um rebase interativo a partir do commit selecionado, carrega em `e`. |
| `` p `` | Escolher | Marque o commit selecionado para ser escolhido (quando meados da base). Isso significa que o commit será mantido ao continuar o rebase. |
| `` F `` | Criar commit de correção | Crie o commit 'correção!' para o commit selecionado. Mais tarde, você pode pressionar `S` neste mesmo commit para aplicar todas os commits de correção acima. |
| `` S `` | Aplicar commits de correções | Aplicar Squash all 'correção!', seja acima do commit selecionado, ou tudo no branch atual (autosquash). |
| `` <ctrl+j>, <alt+down> `` | Mover commit um para baixo |  |
| `` <ctrl+k>, <alt+up> `` | Mover o commit um para cima |  |
| `` V `` | Colar (cherry-pick) |  |
| `` B `` | Marcar como commit base para o rebase | Seleciona um commit base para o próximo rebase. Ao fazer rebase sobre um branch, só são levados os commits acima do commit base. Usa o comando `git rebase --onto`. |
| `` A `` | Modificar | Alterar o commit com mudanças em sted. Se o commit selecionado for o commit HEAD, ele executará o `git commit --amend`. Caso contrário, o compromisso será alterado por meio de uma base de apoio. |
| `` a `` | Alterar atributo de commit | Definir/Redefinir autor de submissão ou co-autor definido. |
| `` t `` | Reverter | Crie um commit reverter para o commit selecionado, que aplica as alterações do commit selecionado em reverso. |
| `` T `` | Etiquetar commit | Cria uma nova tag a apontar para o commit selecionado. É-te pedido o nome da tag e uma descrição opcional. |
| `` <ctrl+l> `` | Ver opções do log | Ver opções do log de commits, p. ex. mudar a ordenação, esconder o grafo do git, mostrar o grafo completo do git. |
| `` G `` | Abrir pull request no browser |  |
| `` <space> `` | Verificar | Faz checkout do commit selecionado como HEAD destacado (detached HEAD). |
| `` y `` | Copiar atributo do commit para a área de transferência | Copia um atributo do commit para a área de transferência (p. ex. hash, URL, diff, mensagem, autor). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Criar novo branch a partir do commit |  |
| `` N `` | Mover commits para uma nova branch | Cria um novo branch e move para lá os commits do branch atual que ainda não foram enviados (push). Útil quando querias começar trabalho novo e te esqueceste de criar primeiro um branch.<br><br>Isto ignora a seleção: o novo branch é sempre criado a partir do branch principal ou empilhado sobre o branch atual (podes escolher). |
| `` w `` | Nova árvore de trabalho |  |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Marcar commit como copiado. Então, dentro da visualização local de commits, você pode pressionar `V` para colar (cherry-pick) o(s) commit(s) copiado(s) em seu branch de check-out. A qualquer momento você pode pressionar `<esc>` para cancelar a seleção. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` * `` | Selecionar os commits do branch atual |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Etiquetas

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar etiqueta para área de transferência |  |
| `` <space> `` | Verificar | Checar a tag selecionada como um HEAD, desanexado |
| `` n `` | Nova etiqueta | Crie uma nova etiqueta a partir do commit atual. Você será solicitado a digitar um nome e uma descrição opcional. |
| `` w `` | Nova árvore de trabalho |  |
| `` d `` | Apagar | Ver opções de exclusão para tag local/remoto. |
| `` P `` | Empurrar etiqueta | Faz push da tag selecionada para um remote. É-te pedido que escolhas o remote. |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Introduzir texto

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar/Cancelar |  |

## Menu

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Executar |  |
| `` <esc> `` | Fechar/Cancelar |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Painel Principal (Normal)

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <mouse wheel down> (fn+up) `` | Rolar para baixo |  |
| `` <mouse wheel up> (fn+down) `` | Rolar para cima |  |
| `` <tab> `` | Mudar de painel de diff | Muda para o outro painel de diff focado. |
| `` a `` | Alternar seleção de hunk | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` v `` | Alternar seleção de intervalo |  |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space> `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` d `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` E `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` <ctrl+o> `` | Copiar as linhas selecionadas do diff para a área de transferência |  |
| `` <left>, h `` | Ir para o local anterior |  |
| `` <right>, l `` | Ir para o próximo trecho |  |
| `` N `` | Ir para o ficheiro anterior |  |
| `` n `` | Ir para o ficheiro seguinte |  |
| `` <ctrl+g> `` | Saltar para ficheiro |  |
| `` G `` | Abrir pull request na linha selecionada | Abre o pull request do branch no browser, na linha onde está a seleção, para poderes comentá-la lá. Só encontra pull requests no GitHub. |
| `` <esc> `` | Voltar ao painel lateral |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` w `` | Fazer commit de alterações sem pré-commit |  |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Painel de confirmação

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar/Cancelar |  |
| `` <ctrl+o> `` | Copiar para a área de transferência |  |

## Painel principal (mesclagem)

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <space> `` | Escolha o local |  |
| `` b `` | Escolher ambos os hunks |  |
| `` <up>, k `` | Trecho anterior |  |
| `` <down>, j `` | Próximo trecho |  |
| `` <left>, h `` | Conflito anterior |  |
| `` <right>, l `` | Próximo conflito |  |
| `` z `` | Desfazer | Desfazer resolução de conflitos de última mesclagem. |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` o `` | Abrir arquivo | Abrir arquivo no aplicativo padrão. |
| `` M `` | Ver opções de conflitos de merge | Ver opções para resolver conflitos de merge. |
| `` <esc> `` | Retornar ao painel de arquivos |  |

## Pull Requests

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Deslocar a vista geral |  |
| `` <space> `` | Começar a rever este pull request |  |
| `` <enter> `` | Abrir o pull request em modo review |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Reflog

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar hash abreviado do commit para a área de transferência |  |
| `` <space> `` | Verificar | Faz checkout do commit selecionado como HEAD destacado (detached HEAD). |
| `` y `` | Copiar atributo do commit para a área de transferência | Copia um atributo do commit para a área de transferência (p. ex. hash, URL, diff, mensagem, autor). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Criar novo branch a partir do commit |  |
| `` N `` | Mover commits para uma nova branch | Cria um novo branch e move para lá os commits do branch atual que ainda não foram enviados (push). Útil quando querias começar trabalho novo e te esqueceste de criar primeiro um branch.<br><br>Isto ignora a seleção: o novo branch é sempre criado a partir do branch principal ou empilhado sobre o branch atual (podes escolher). |
| `` w `` | Nova árvore de trabalho |  |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Marcar commit como copiado. Então, dentro da visualização local de commits, você pode pressionar `V` para colar (cherry-pick) o(s) commit(s) copiado(s) em seu branch de check-out. A qualquer momento você pode pressionar `<esc>` para cancelar a seleção. |
| `` <ctrl+r> `` | Limpar a seleção de commits copiados (cherry-pick) |  |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` * `` | Selecionar os commits do branch atual |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver commits |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Remotes

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Ver branches |  |
| `` n `` | Novo controle |  |
| `` d `` | Remover | Remover o controle remoto. Quaisquer ramificações locais de rastreamento de um ramo remoto do controle não serão afetadas. |
| `` e `` | Editar | Edita o nome ou o URL do remote selecionado. |
| `` f `` | Buscar | Faz fetch de atualizações do repositório remote. Obtém novos commits e branches sem os juntar aos teus branches locais. |
| `` F `` | Adicionar remote de fork | Adiciona rapidamente um remote de fork, substituindo o dono no URL do origin, e opcionalmente faz checkout de um branch do novo remote. |
| `` / `` | Filtrar a visualização atual por texto |  |

## Review do PR

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Ver diff |  |
| `` <space> `` | Marcar/desmarcar como revisto |  |
| `` t `` | Alternar diff unificado / lado a lado |  |
| `` d `` | Ver descrição do PR |  |
| `` G `` | Mostrar/esconder ficheiros gerados |  |
| `` R `` | Atualizar pull request |  |
| `` S `` | Submeter review |  |
| `` 0 `` | Focar visualização principal |  |
| `` / `` | Filtrar a visualização atual por texto |  |
| `` <space> `` | Selecionar intervalo para baixo |  |
| `` c `` | Adicionar comentário de review |  |
| `` t `` | Resolver/reabrir a thread selecionada |  |
| `` C `` | Adicionar comentário à review pendente |  |
| `` <esc> `` | Retornar ao painel de arquivos |  |

## Secundário

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <tab> `` | Mudar de painel de diff | Muda para o outro painel de diff focado. |
| `` a `` | Alternar seleção de hunk | Ativa/desativa modo linha por linha vs. modo de seleção por partes. |
| `` v `` | Alternar seleção de intervalo |  |
| `` e `` | Editar arquivo | Abrir arquivo no editor externo. |
| `` <space> `` | Etapa | Ativar/desativar seleção em staged/unstaged |
| `` d `` | Descartar | Quando a mudança não desejada for selecionada, descarte a mudança usando `git reset`. Quando a mudança em fase é selecionada, despare a mudança. |
| `` E `` | Editar hunk | Editar o local selecionado no editor externo. |
| `` <ctrl+o> `` | Copiar as linhas selecionadas do diff para a área de transferência |  |
| `` <left>, h `` | Ir para o local anterior |  |
| `` <right>, l `` | Ir para o próximo trecho |  |
| `` N `` | Ir para o ficheiro anterior |  |
| `` n `` | Ir para o ficheiro seguinte |  |
| `` <ctrl+g> `` | Saltar para ficheiro |  |
| `` G `` | Abrir pull request na linha selecionada | Abre o pull request do branch no browser, na linha onde está a seleção, para poderes comentá-la lá. Só encontra pull requests no GitHub. |
| `` <esc> `` | Voltar ao painel lateral |  |
| `` c `` | Commit | Submeter mudanças em staging |
| `` w `` | Fazer commit de alterações sem pré-commit |  |
| `` C `` | Enviar alteração usando um editor Git |  |
| `` <ctrl+f> `` | Encontrar commit da base para corrigir | Encontre o commit em que as suas mudanças atuais estão se baseando, para alterar/consertar o commit. Isso poupa-te você de ter que olhar pelos commits da sua branch um por um para ver qual commit deve ser alterado/consertado<br>Veja a documentação:<br><https://github.com/jesseduffield/lazygit/tree/master/docs/Fixup_Commits.md> |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Stash

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <space> `` | Aplicar | Aplique o stash no seu diretório de trabalho. |
| `` g `` | Pop | Aplique a entrada de stash no seu diretório de trabalho e remova a entrada de stash. |
| `` d `` | Descartar | Remova a entrada do stash da lista de armazenamento. |
| `` n `` | Nova branch | Criar um novo ramo a partir da entrada de lixo selecionada. Isso funciona verificando o commit do qual a entrada de lixo foi criada, criar um novo branch a partir desse commit e, em seguida, aplicar a entrada de lixo ao novo branch como um commit adicional. |
| `` w `` | Nova árvore de trabalho |  |
| `` r `` | Renomear o stash |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Status

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` e `` | Editar arquivo de configuração | Abrir arquivo no editor externo. |
| `` u `` | Verificar atualização |  |
| `` <enter> `` | Mudar para um repositório recente |  |
| `` a `` | Mostrar/ciclo todos os logs de filiais |  |
| `` A `` | Mostrar/alternar logs de todos os branches (inverso) |  |
| `` 0 `` | Focar visualização principal |  |

## Sub-commits

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar hash abreviado do commit para a área de transferência |  |
| `` <space> `` | Verificar | Faz checkout do commit selecionado como HEAD destacado (detached HEAD). |
| `` y `` | Copiar atributo do commit para a área de transferência | Copia um atributo do commit para a área de transferência (p. ex. hash, URL, diff, mensagem, autor). |
| `` o `` | Abrir commit no navegador |  |
| `` n `` | Criar novo branch a partir do commit |  |
| `` N `` | Mover commits para uma nova branch | Cria um novo branch e move para lá os commits do branch atual que ainda não foram enviados (push). Útil quando querias começar trabalho novo e te esqueceste de criar primeiro um branch.<br><br>Isto ignora a seleção: o novo branch é sempre criado a partir do branch principal ou empilhado sobre o branch atual (podes escolher). |
| `` w `` | Nova árvore de trabalho |  |
| `` g `` | Restaurar | Ver opções de redefinição (soft/mixed/hard) para redefinir para o item selecionado. |
| `` C `` | Copiar (cherry-pick) | Marcar commit como copiado. Então, dentro da visualização local de commits, você pode pressionar `V` para colar (cherry-pick) o(s) commit(s) copiado(s) em seu branch de check-out. A qualquer momento você pode pressionar `<esc>` para cancelar a seleção. |
| `` <ctrl+r> `` | Limpar a seleção de commits copiados (cherry-pick) |  |
| `` <ctrl+t> `` | Abrir ferramenta de diff externa (git difftool) |  |
| `` * `` | Selecionar os commits do branch atual |  |
| `` 0 `` | Focar visualização principal |  |
| `` <enter> `` | Ver arquivos |  |
| `` / `` | Pesquisar na visualização atual por texto |  |

## Submódulos

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <ctrl+o> `` | Copiar o nome do submódulo para área de transferência |  |
| `` <enter> `` | Entrar | Entra no submódulo. Depois de entrar, carrega em `<esc>` para voltar ao repositório principal. |
| `` d `` | Remover | Remova o submódulo selecionado e o diretório correspondente. |
| `` u `` | Atualizar | Atualizar submódulo selecionado. |
| `` n `` | Novo submódulo |  |
| `` e `` | Atualizar URL do submódulo |  |
| `` i `` | Inicializar | Inicializa o submódulo selecionado para preparar o fetch. Normalmente, a seguir, vais querer usar a ação 'update' para fazer fetch do submódulo. |
| `` b `` | Ver opções de submódulos em massa |  |
| `` / `` | Filtrar a visualização atual por texto |  |

## Sumário do commit

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` <enter> `` | Confirmar |  |
| `` <esc> `` | Fechar |  |

## Árvores de trabalho

| Tecla | Ação | Info |
|-----|--------|-------------|
| `` n `` | Nova árvore de trabalho |  |
| `` <space> `` | Mudar | Mudar para a árvore de trabalho selecionada. |
| `` o `` | Abrir no editor |  |
| `` d `` | Remover | Remove o worktree selecionado. Apaga tanto o diretório do worktree como os metadados sobre o worktree no diretório .git. |
| `` / `` | Filtrar a visualização atual por texto |  |

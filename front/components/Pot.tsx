import { Game, Pot as PotType } from "../interfaces/index";
import { useEffect, useRef, useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";

const initialPot: PotType[] = [
    {
        topShare: 0,
        amount: 0,
        eligiblePlayerNums: [],
        winningPlayerNums: [],
        winningHand: [],
        winningScore: 0,
    },
];

export default function Pot() {
    const { appState, dispatch } = useContext(AppContext);
    const [stage, setStage] = useState(2);
    const [pots, setPots] = useState(initialPot);

    const game = appState.game;

    useEffect(() => {
        if (game && game.stage !== undefined) {
            if (game.stage !== stage) {
                setStage(game.stage);

                if (game.stage === 2) {
                    setPots(initialPot);
                } else if (game.pots) {
                    setPots(game.pots);
                }
            }
        }
    }, [game?.stage, game?.pots, stage]);

    if (!game || !game.pots) {
        return null;
    }

    return (
        <div>
            {pots.map((pot, index) => (
                <div className="flex flex-col" key={index}>
                    <div className="flex w-full justify-end">
                        {game.pots[index] && game.pots[index].amount != pot.amount ? (
                            <p className="text-sm font-normal text-white">
                                total: {game.pots[index].amount}
                            </p>
                        ) : (
                            <p className="text-sm font-normal text-white">&nbsp;</p>
                        )}
                    </div>
                    <p className="flex h-10 w-24 flex-col items-center justify-center rounded-3xl bg-green-900 text-2xl font-semibold text-white ">
                        {pot.amount}
                    </p>
                </div>
            ))}
        </div>
    );
}

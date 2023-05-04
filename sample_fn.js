module.exports = async function (context) {
    let primes = [];
    let num = 2;

    while (primes.length < 100) {
        let isPrime = true;

        for (let i = 2; i <= Math.sqrt(num); i++) {
            if (num % i === 0) {
                isPrime = false;
                break;
            }
        }

        if (isPrime) {
            primes.push(num);
        }

        num++;
    }
    return {
        status: 200,
        body: "primes generated!\n"
    };
}